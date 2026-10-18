package book

import (
	"slices"

	"orderbook/levels"
)

type side struct {
	levels []levels.LevelIndex
	bid    bool
}

func (s *side) find(lv *levels.Levels, price uint64) (int, bool) {
	n := len(s.levels)
	if n == 0 {
		return 0, false
	}

	r := n - 1
	rPrice := lv.Price(s.levels[r])
	if rPrice == price {
		return r, true
	}
	if (s.bid && rPrice < price) ||
		(!s.bid && rPrice > price) {
		return n, false
	}

	l, step := r, 1
	for l >= 0 {
		lPrice := lv.Price(s.levels[l])
		if (s.bid && lPrice <= price) || (!s.bid && lPrice >= price) {
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
		mPrice := lv.Price(s.levels[m])
		if mPrice == price {
			return m, true
		}

		if (s.bid && mPrice > price) ||
			(!s.bid && mPrice < price) {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}

func (s *side) insertAt(i int, idx levels.LevelIndex) {
	s.levels = slices.Insert(s.levels, i, idx)
}

func (s *side) removeAt(i int) {
	s.levels = append(s.levels[:i], s.levels[i+1:]...)
}

func (s *side) best() (levels.LevelIndex, bool) {
	if len(s.levels) == 0 {
		return -1, false
	}
	return s.levels[len(s.levels)-1], true
}
