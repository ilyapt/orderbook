package matching

import (
	"errors"

	"orderbook/book"
	"orderbook/levels"
)

var (
	ErrZeroQty    = errors.New("order quantity must be greater than zero")
	ErrZeroPrice  = errors.New("order price must be greater than zero")
	ErrInvalidTIF = errors.New("incorrect order tif")
)

type Engine struct {
	book        *book.Book
	lastOrderID uint64
	lastTradeID uint64
}

func New(ordersCap, levelsCap int) *Engine {
	return &Engine{
		book: book.New(ordersCap, levelsCap),
	}
}

func (e *Engine) Place(order Order) (Report, error) {
	if order.Qty == 0 {
		return Report{}, ErrZeroQty
	}
	if order.Price == 0 {
		return Report{}, ErrZeroPrice
	}
	if order.TIF < GTC || order.TIF > PostOnly {
		return Report{}, ErrInvalidTIF
	}

	e.lastOrderID++
	report := Report{
		OrderID:      uint64(levels.SeqHandle(e.lastOrderID)),
		RemainingQty: order.Qty,
	}
	opp := order.Side.Opposite()

	switch order.TIF {
	case PostOnly:
		if price, _, ok := e.book.LevelAt(opp, 0); ok && crossed(order, price) {
			report.Status = Rejected
			return report, nil
		}
	case FOK:
		var sum uint64
		for depth := 0; sum < order.Qty; depth++ {
			price, totalQty, ok := e.book.LevelAt(opp, depth)
			if !ok || !crossed(order, price) {
				report.Status = Rejected
				return report, nil
			}
			sum += totalQty
		}
	}

	for report.RemainingQty > 0 {
		h, price, qty, ok := e.book.Front(opp)
		if !ok || !crossed(order, price) {
			break
		}
		take := min(report.RemainingQty, qty)
		remaining := e.book.Fill(h, take)
		e.lastTradeID++
		report.Trades = append(report.Trades, Trade{
			ID:             e.lastTradeID,
			MakerOrderID:   uint64(h),
			Price:          price,
			Qty:            take,
			MakerRemaining: remaining,
		})
		report.RemainingQty -= take
	}

	switch {
	case report.RemainingQty == 0:
		report.Status = Filled
	case order.TIF == IOC:
		report.Status = Canceled
	default:
		h, err := e.book.Add(order.Side, e.lastOrderID, order.Price, report.RemainingQty)
		if err != nil {
			return Report{}, err
		}
		report.OrderID = uint64(h)
		if len(report.Trades) == 0 {
			report.Status = Open
		} else {
			report.Status = PartiallyFilled
		}
	}
	return report, nil
}

func (e *Engine) Cancel(orderID uint64) (Report, error) {
	remaining, err := e.book.Cancel(levels.Handle(orderID))
	if err != nil {
		return Report{}, err
	}
	return Report{
		OrderID:      orderID,
		RemainingQty: remaining,
		Status:       Canceled,
	}, nil
}

func crossed(order Order, oppositePrice uint64) bool {
	if order.Side == book.Sell {
		return oppositePrice >= order.Price
	}
	return oppositePrice <= order.Price
}
