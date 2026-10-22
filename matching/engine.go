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
	sink        TradeSink
	lastOrderID uint64
	lastTradeID uint64
}

func New(ordersCap, levelsCap int, sink TradeSink) (*Engine, error) {
	if sink == nil {
		return nil, errors.New("sink cannot be nil")
	}
	return &Engine{
		book: book.New(ordersCap, levelsCap),
		sink: sink,
	}, nil
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
	taker := e.book.Reserve(e.lastOrderID)
	report := Report{
		OrderID:      uint64(taker),
		RemainingQty: order.Qty,
	}
	opp := order.Side.Opposite()

	switch order.TIF {
	case PostOnly:
		if price, _, ok := e.book.LevelAt(opp, 0); ok && crossed(order, price) {
			e.book.Release(taker)
			report.Status = Rejected
			return report, nil
		}
	case FOK:
		var available uint64
		for depth := 0; available < order.Qty; depth++ {
			price, totalQty, ok := e.book.LevelAt(opp, depth)
			if !ok || !crossed(order, price) {
				e.book.Release(taker)
				report.Status = Rejected
				return report, nil
			}
			available += totalQty
		}
	}

	for report.RemainingQty > 0 {
		maker, price, makerQty, ok := e.book.Front(opp)
		if !ok || !crossed(order, price) {
			break
		}
		take := min(report.RemainingQty, makerQty)
		makerRemaining := e.book.Fill(maker, take)
		e.lastTradeID++
		report.RemainingQty -= take

		e.sink.OnTrade(Trade{
			ID:             e.lastTradeID,
			TakerOrderID:   uint64(taker),
			MakerOrderID:   uint64(maker),
			Price:          price,
			Qty:            take,
			TakerRemaining: report.RemainingQty,
			MakerRemaining: makerRemaining,
		})
	}

	switch {
	case report.RemainingQty == 0:
		e.book.Release(taker)
		report.Status = Filled
	case order.TIF == IOC:
		e.book.Release(taker)
		report.Status = Canceled
	default:
		if err := e.book.Rest(order.Side, taker, order.Price, report.RemainingQty); err != nil {
			panic(err)
		}
		if report.RemainingQty == order.Qty {
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
