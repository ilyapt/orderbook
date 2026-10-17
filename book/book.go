package book

import "orderbook/levels"

type Side int8

const (
	Buy Side = iota + 1
	Sell
)

type Book struct {
	levels *levels.Levels
	bids   side
	asks   side
}

func New(ordersCap, levelsCap int) *Book {
	return &Book{
		levels: levels.New(ordersCap, levelsCap),
		bids:   side{bid: true},
		asks:   side{bid: false},
	}
}

func (b *Book) Add(s Side, price, qty uint64) (levels.Handle, error) {
	if qty == 0 {
		return levels.NoHandle, levels.ErrZeroQty
	}
	sd := b.side(s)

	i, found := sd.find(price)
	if found {
		return b.levels.Push(sd.levels[i].lvl, qty)
	}

	lvl := b.levels.NewLevel(price)
	sd.insertAt(i, price, lvl)
	return b.levels.Push(lvl, qty)
}

func (b *Book) side(s Side) *side {
	switch s {
	case Buy:
		return &b.bids
	case Sell:
		return &b.asks
	}
	panic("book: invalid side")
}

func (b *Book) Cancel(h levels.Handle) error {
	idx, empty, err := b.levels.Cancel(h)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}

	price := b.levels.Price(idx)

	i, found := b.bids.find(price)
	if found && b.bids.levels[i].lvl == idx {
		b.bids.removeAt(i)
		b.levels.FreeLevel(idx)
		return nil
	}

	i, found = b.asks.find(price)
	if found && b.asks.levels[i].lvl == idx {
		b.asks.removeAt(i)
		b.levels.FreeLevel(idx)
		return nil
	}

	panic("book: level missing from side")
}

func (b *Book) Best(s Side) (price, totalQty uint64, ok bool) {
	l, ok := b.side(s).best()

	if !ok {
		return 0, 0, false
	}

	return l.price, b.levels.TotalQty(l.lvl), true
}
