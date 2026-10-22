package book

import "orderbook/levels"

type Side int8

const (
	Buy Side = iota + 1
	Sell
)

func (s Side) Opposite() Side {
	if s == Buy {
		return Sell
	}
	return Buy
}

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
func (b *Book) Reserve(seq uint64) levels.Handle {
	return b.levels.Reserve(seq)
}

func (b *Book) Rest(s Side, h levels.Handle, price, qty uint64) error {
	if qty == 0 {
		b.levels.Release(h)
		return levels.ErrZeroQty
	}
	sd := b.side(s)

	i, found := sd.find(b.levels, price)
	if found {
		b.levels.Rest(h, sd.levels[i], qty)
		return nil
	}

	lvl := b.levels.NewLevel(price)
	sd.insertAt(i, lvl)
	b.levels.Rest(h, lvl, qty)
	return nil
}

func (b *Book) Release(h levels.Handle) {
	b.levels.Release(h)
}

func (b *Book) Add(s Side, seq, price, qty uint64) (levels.Handle, error) {
	h := b.Reserve(seq)
	if err := b.Rest(s, h, price, qty); err != nil {
		return levels.NoHandle, err
	}
	return h, nil
}

func (b *Book) LevelAt(s Side, depth int) (price, totalQty uint64, ok bool) {
	sd := b.side(s)
	i := len(sd.levels) - 1 - depth
	if i < 0 {
		return 0, 0, false
	}
	l := sd.levels[i]
	return b.levels.Price(l), b.levels.TotalQty(l), true
}

func (b *Book) Cancel(h levels.Handle) (uint64, error) {
	idx, remaining, empty, err := b.levels.Cancel(h)
	if err != nil {
		return 0, err
	}
	if empty {
		b.dropLevel(idx)
	}

	return remaining, nil
}

func (b *Book) Best(s Side) (price, totalQty uint64, ok bool) {
	l, ok := b.side(s).best()

	if !ok {
		return 0, 0, false
	}

	return b.levels.Price(l), b.levels.TotalQty(l), true
}

func (b *Book) Front(s Side) (h levels.Handle, price, qty uint64, ok bool) {
	l, ok := b.side(s).best()

	if !ok {
		return levels.NoHandle, 0, 0, false
	}

	h, qty, ok = b.levels.Front(l)
	if !ok {
		panic("book: empty level")
	}

	return h, b.levels.Price(l), qty, true
}

func (b *Book) Fill(h levels.Handle, qty uint64) (remaining uint64) {
	l, remaining, empty := b.levels.Fill(h, qty)
	if empty {
		b.dropLevel(l)

	}
	return remaining
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

func (b *Book) dropLevel(idx levels.LevelIndex) {
	price := b.levels.Price(idx)

	i, found := b.bids.find(b.levels, price)
	if found && b.bids.levels[i] == idx {
		b.bids.removeAt(i)
		b.levels.FreeLevel(idx)
		return
	}

	i, found = b.asks.find(b.levels, price)
	if found && b.asks.levels[i] == idx {
		b.asks.removeAt(i)
		b.levels.FreeLevel(idx)
		return
	}

	panic("book: level missing from side")
}
