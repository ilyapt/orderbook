package book

import (
	"orderbook/levels"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeLevels() (*levels.Levels, []levels.LevelIndex) {
	l := levels.New(100, 100)
	l.NewLevel(0)
	l.NewLevel(0)
	l6 := l.NewLevel(6)
	l.NewLevel(0)
	l8 := l.NewLevel(8)
	l2 := l.NewLevel(2)
	l.NewLevel(0)
	l.NewLevel(0)
	l.NewLevel(0)
	l3 := l.NewLevel(3)
	l.NewLevel(0)
	l.NewLevel(0)
	l7 := l.NewLevel(7)
	l.NewLevel(0)
	l4 := l.NewLevel(4)
	return l, []levels.LevelIndex{l2, l3, l4, l6, l7, l8}
}

func TestBidFind(t *testing.T) {
	l, lvls := makeLevels()
	s := side{
		bid:    true,
		levels: lvls,
	}

	idx, ok := s.find(l, 4)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	idx, ok = s.find(l, 5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(l, 6)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(l, 1)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	idx, ok = s.find(l, 8)
	require.True(t, ok)
	require.Equal(t, 5, idx)

	idx, ok = s.find(l, 9)
	require.False(t, ok)
	require.Equal(t, 6, idx)
}

func TestAskFind(t *testing.T) {
	l, lvls := makeLevels()
	slices.Reverse(lvls)
	s := side{
		bid:    false,
		levels: lvls,
	}

	idx, ok := s.find(l, 4)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(l, 5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(l, 6)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	idx, ok = s.find(l, 1)
	require.False(t, ok)
	require.Equal(t, 6, idx)

	idx, ok = s.find(l, 8)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	idx, ok = s.find(l, 9)
	require.False(t, ok)
	require.Equal(t, 0, idx)
}

func TestBidInsert(t *testing.T) {
	l, lvls := makeLevels()
	l2, l3, l4, l6, l7, l8 := lvls[0], lvls[1], lvls[2], lvls[3], lvls[4], lvls[5]
	s := side{
		bid:    true,
		levels: slices.Grow(lvls, 1),
	}

	l5 := l.NewLevel(5)
	idx, ok := s.find(l, 5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, l5)
	require.Equal(t, []levels.LevelIndex{l2, l3, l4, l5, l6, l7, l8}, s.levels)

	l1 := l.NewLevel(1)
	idx, ok = s.find(l, 1)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, l1)
	require.Equal(t, []levels.LevelIndex{l1, l2, l3, l4, l5, l6, l7, l8}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, l8, best)

	l9 := l.NewLevel(9)
	idx, ok = s.find(l, 9)
	require.False(t, ok)
	require.Equal(t, 8, idx)

	s.insertAt(idx, l9)
	require.Equal(t, []levels.LevelIndex{l1, l2, l3, l4, l5, l6, l7, l8, l9}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, l9, best)
}

func TestAskInsert(t *testing.T) {
	l, lvls := makeLevels()
	l2, l3, l4, l6, l7, l8 := lvls[0], lvls[1], lvls[2], lvls[3], lvls[4], lvls[5]
	slices.Reverse(lvls)
	s := side{
		bid:    false,
		levels: slices.Grow(lvls, 1),
	}

	l5 := l.NewLevel(5)
	idx, ok := s.find(l, 5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, l5)
	require.Equal(t, []levels.LevelIndex{l8, l7, l6, l5, l4, l3, l2}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, l2, best)

	l1 := l.NewLevel(1)
	idx, ok = s.find(l, 1)
	require.False(t, ok)
	require.Equal(t, 7, idx)

	s.insertAt(idx, l1)
	require.Equal(t, []levels.LevelIndex{l8, l7, l6, l5, l4, l3, l2, l1}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, l1, best)

	l9 := l.NewLevel(9)
	idx, ok = s.find(l, 9)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, l9)
	require.Equal(t, []levels.LevelIndex{l9, l8, l7, l6, l5, l4, l3, l2, l1}, s.levels)
}

func TestBidRemove(t *testing.T) {
	l, lvls := makeLevels()
	l2, l3, l6, l7, l8 := lvls[0], lvls[1], lvls[3], lvls[4], lvls[5]
	s := side{
		bid:    true,
		levels: lvls,
	}

	idx, ok := s.find(l, 4)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l2, l3, l6, l7, l8}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, l8, best)

	idx, ok = s.find(l, 2)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l3, l6, l7, l8}, s.levels)

	idx, ok = s.find(l, 8)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l3, l6, l7}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, l7, best)
}

func TestAskRemove(t *testing.T) {
	l, lvls := makeLevels()
	l2, l3, l6, l7, l8 := lvls[0], lvls[1], lvls[3], lvls[4], lvls[5]
	slices.Reverse(lvls)
	s := side{
		bid:    false,
		levels: lvls,
	}

	idx, ok := s.find(l, 4)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l8, l7, l6, l3, l2}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, l2, best)

	idx, ok = s.find(l, 2)
	require.True(t, ok)
	require.Equal(t, 4, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l8, l7, l6, l3}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, l3, best)

	idx, ok = s.find(l, 8)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Equal(t, []levels.LevelIndex{l7, l6, l3}, s.levels)
}

func TestEmpty(t *testing.T) {
	_, lvls := makeLevels()
	l7 := lvls[4]
	s := side{bid: true}

	best, found := s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)

	s.insertAt(0, l7)
	require.Equal(t, []levels.LevelIndex{l7}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, l7, best)

	s.removeAt(0)
	require.Empty(t, s.levels)

	best, found = s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)
}
