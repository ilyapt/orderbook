package book

import (
	"slices"

	"orderbook/levels"
)

type level struct {
	price uint64
	lvl   levels.LevelIndex
}

type side struct {
	levels []level
	bid    bool
}

func (s *side) find(price uint64) (int, bool) {
	n := len(s.levels)
	if n == 0 {
		return 0, false
	}

	r := n - 1
	if s.levels[r].price == price {
		return r, true
	}
	if (s.bid && s.levels[r].price < price) ||
		(!s.bid && s.levels[r].price > price) {
		return n, false
	}

	l, step := r, 1
	for l >= 0 && ((s.bid && s.levels[l].price > price) ||
		(!s.bid && s.levels[l].price < price)) {
		r = l
		l -= step
		step <<= 1
	}
	if l < 0 {
		l = 0
	}

	for l <= r {
		m := l + (r-l)/2
		if s.levels[m].price == price {
			return m, true
		}

		if (s.bid && s.levels[m].price > price) ||
			(!s.bid && s.levels[m].price < price) {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}

func (s *side) insertAt(i int, price uint64, idx levels.LevelIndex) {
	s.levels = slices.Insert(s.levels, i, level{price: price, lvl: idx})
}

func (s *side) removeAt(i int) {
	s.levels = append(s.levels[:i], s.levels[i+1:]...)
}

func (s *side) best() (level, bool) {
	if len(s.levels) == 0 {
		return level{}, false
	}
	return s.levels[len(s.levels)-1], true
}
