package book

import (
	"slices"

	"orderbook/levels"
)

type entry struct {
	price uint64
	idx   int32
}

type side struct {
	levels []entry
	bid    bool
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

func (s *side) find(price uint64) (int, bool) {
	return gallop(s.levels, s.bid, price)
}

func (s *side) insertAt(i int, idx levels.LevelIndex, price uint64) {
	s.levels = slices.Insert(s.levels, i, entry{price: price, idx: int32(idx)})
}

func (s *side) removeAt(i int) {
	s.levels = append(s.levels[:i], s.levels[i+1:]...)
}

func (s *side) best() (levels.LevelIndex, bool) {
	if len(s.levels) == 0 {
		return -1, false
	}
	return levels.LevelIndex(s.levels[len(s.levels)-1].idx), true
}
