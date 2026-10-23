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

	p, l, found := sd.find(price)
	if found {
		b.levels.Rest(h, l, qty)
		return nil
	}

	lvl := b.levels.NewLevel(price)
	b.levels.Rest(h, lvl, qty)
	sd.insertAt(p, lvl, price)
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

type Cursor struct {
	b  *Book
	cu cursor
	ok bool
}

func (b *Book) Cursor(s Side) Cursor {
	cu, ok := b.side(s).cursor()
	return Cursor{b: b, cu: cu, ok: ok}
}

func (c *Cursor) Next() (price, totalQty uint64, ok bool) {
	if !c.ok {
		return 0, 0, false
	}
	l := c.cu.idx()
	c.ok = c.cu.prev()
	return c.b.levels.Price(l), c.b.levels.TotalQty(l), true
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

	sd, other := &b.asks, &b.bids
	if n := len(b.bids.chain); n > 0 && price <= b.bids.chain[n-1].price {
		sd, other = &b.bids, &b.asks
	}

	p, l, found := sd.find(price)
	if found && l == idx {
		sd.removeAt(p)
		b.levels.FreeLevel(idx)
		return
	}

	p, l, found = other.find(price)
	if found && l == idx {
		other.removeAt(p)
		b.levels.FreeLevel(idx)
		return
	}

	panic("book: level missing from side")
}
