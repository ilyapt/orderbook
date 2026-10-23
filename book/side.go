package book

import (
	"slices"

	"orderbook/levels"
)

const blockCap = 128

type block struct {
	n       int32
	entries [blockCap]entry
}

type entry struct {
	price uint64
	idx   int32
}

type side struct {
	blocks []block
	chain  []entry
	bid    bool
}

type pos struct {
	c, i int
}

func gallop(s []entry, bid bool, price uint64) (int, bool) {
	n := len(s)
	if n == 0 {
		return 0, false
	}

	r := n - 1
	rPrice := s[r].price
	if rPrice == price {
		return r, true
	}
	if (bid && rPrice < price) ||
		(!bid && rPrice > price) {
		return n, false
	}

	l, step := r, 1
	for l >= 0 {
		lPrice := s[l].price
		if (bid && lPrice <= price) || (!bid && lPrice >= price) {
			break
		}
		r = l
		l -= step
		step <<= 1
	}
	if l < 0 {
		l = 0
	}

	for l <= r {
		m := l + (r-l)/2
		mPrice := s[m].price
		if mPrice == price {
			return m, true
		}

		if (bid && mPrice > price) ||
			(!bid && mPrice < price) {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}

func (s *side) find(price uint64) (pos, bool) {
	if len(s.chain) == 0 {
		return pos{0, 0}, false
	}

	c := len(s.chain) - 1
	if key := s.chain[c].price; (s.bid && price < key) || (!s.bid && price > key) {
		var found bool
		c, found = gallop(s.chain, s.bid, price)
		if found {
			return pos{c, 0}, true
		}
		if c > 0 {
			c--
		}
	}

	b := &s.blocks[s.chain[c].idx]
	i, found := gallop(b.entries[:b.n], s.bid, price)
	return pos{c, i}, found
}

func (s *side) insertAt(p pos, idx levels.LevelIndex, price uint64) {
	if len(s.chain) == 0 {
		s.blocks = append(s.blocks, block{n: 1, entries: [blockCap]entry{{price: price, idx: int32(idx)}}})
		s.chain = append(s.chain, entry{price: price, idx: int32(len(s.blocks) - 1)})
		return
	}

	k := s.chain[p.c].idx
	if s.blocks[k].n == blockCap {
		m := int32(blockCap / 2)
		s.blocks = append(s.blocks, block{n: blockCap - m})
		nk := int32(len(s.blocks) - 1)
		copy(s.blocks[nk].entries[:], s.blocks[k].entries[m:])
		s.blocks[k].n = m
		s.chain = slices.Insert(s.chain, p.c+1, entry{price: s.blocks[nk].entries[0].price, idx: nk})
		if p.i > int(m) {
			p.c++
			p.i -= int(m)
			k = nk
		}
	}

	b := &s.blocks[k]
	copy(b.entries[p.i+1:b.n+1], b.entries[p.i:b.n])
	b.entries[p.i] = entry{price: price, idx: int32(idx)}
	b.n++
	if p.i == 0 {
		s.chain[p.c].price = price
	}
}

func (s *side) removeAt(p pos) {
	k := s.chain[p.c].idx
	b := &s.blocks[k]
	copy(b.entries[p.i:b.n-1], b.entries[p.i+1:b.n])
	b.n--

	if b.n == 0 {
		s.chain = append(s.chain[:p.c], s.chain[p.c+1:]...)
		last := int32(len(s.blocks) - 1)
		if k != last {
			s.blocks[k] = s.blocks[last]
			c, _ := gallop(s.chain, s.bid, s.blocks[k].entries[0].price)
			s.chain[c].idx = k
		}
		s.blocks = s.blocks[:last]
		return
	}
	if p.i == 0 {
		s.chain[p.c].price = b.entries[0].price
	}
}

func (s *side) at(p pos) levels.LevelIndex {
	return levels.LevelIndex(s.blocks[s.chain[p.c].idx].entries[p.i].idx)
}

func (s *side) atDepth(depth int) (levels.LevelIndex, bool) {
	for c := len(s.chain) - 1; c >= 0; c-- {
		b := &s.blocks[s.chain[c].idx]
		if depth < int(b.n) {
			return levels.LevelIndex(b.entries[int(b.n)-1-depth].idx), true
		}
		depth -= int(b.n)
	}
	return -1, false
}

func (s *side) depth(out []levels.LevelIndex) int {
	d := 0
	for c := len(s.chain) - 1; c >= 0 && d < len(out); c-- {
		b := &s.blocks[s.chain[c].idx]
		es := b.entries[:b.n]
		for j := len(es) - 1; j >= 0 && d < len(out); j-- {
			out[d] = levels.LevelIndex(es[j].idx)
			d++
		}
	}
	return d
}

type cursor struct {
	s *side
	b *block
	c int
	i int
}

func (s *side) cursor() (cursor, bool) {
	c := len(s.chain) - 1
	if c < 0 {
		return cursor{}, false
	}
	b := &s.blocks[s.chain[c].idx]
	return cursor{s: s, b: b, c: c, i: int(b.n) - 1}, true
}

func (cu *cursor) idx() levels.LevelIndex {
	return levels.LevelIndex(cu.b.entries[cu.i].idx)
}

func (cu *cursor) prev() bool {
	if cu.i > 0 {
		cu.i--
		return true
	}
	if cu.c == 0 {
		return false
	}
	cu.c--
	cu.b = &cu.s.blocks[cu.s.chain[cu.c].idx]
	cu.i = int(cu.b.n) - 1
	return true
}

func (s *side) best() (levels.LevelIndex, bool) {
	if len(s.chain) == 0 {
		return -1, false
	}
	c := s.chain[len(s.chain)-1]
	b := &s.blocks[c.idx]
	return levels.LevelIndex(b.entries[b.n-1].idx), true
}
