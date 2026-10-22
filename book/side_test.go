package book

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/levels"
)

func TestBidFind(t *testing.T) {
	s := side{
		bid: true,
		levels: []entry{
			{price: 2, idx: 10},
			{price: 3, idx: 12},
			{price: 4, idx: 37},
			{price: 6, idx: 99},
			{price: 7, idx: 45},
			{price: 8, idx: 18},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	idx, ok = s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(6)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 5, idx)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 6, idx)
}

func TestAskFind(t *testing.T) {
	s := side{
		bid: false,
		levels: []entry{
			{price: 8, idx: 18},
			{price: 7, idx: 45},
			{price: 6, idx: 99},
			{price: 4, idx: 37},
			{price: 3, idx: 12},
			{price: 2, idx: 10},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	idx, ok = s.find(6)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 6, idx)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 0, idx)
}

func TestBidInsert(t *testing.T) {
	s := side{
		bid: true,
		levels: slices.Grow([]entry{
			{price: 2, idx: 10},
			{price: 3, idx: 12},
			{price: 4, idx: 37},
			{price: 6, idx: 99},
			{price: 7, idx: 45},
			{price: 8, idx: 18},
		}, 1),
	}

	idx, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, 67, 5)
	require.Equal(t, []entry{
		{2, 10},
		{3, 12},
		{4, 37},
		{5, 67},
		{6, 99},
		{7, 45},
		{8, 18},
	}, s.levels)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, 81, 1)
	require.Equal(t, []entry{
		{1, 81},
		{2, 10},
		{3, 12},
		{4, 37},
		{5, 67},
		{6, 99},
		{7, 45},
		{8, 18},
	}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(18), best)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 8, idx)

	s.insertAt(idx, 22, 9)
	require.Equal(t, []entry{
		{1, 81},
		{2, 10},
		{3, 12},
		{4, 37},
		{5, 67},
		{6, 99},
		{7, 45},
		{8, 18},
		{9, 22},
	}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(22), best)
}

func TestAskInsert(t *testing.T) {
	s := side{
		bid: false,
		levels: slices.Grow([]entry{
			{price: 8, idx: 18},
			{price: 7, idx: 45},
			{price: 6, idx: 99},
			{price: 4, idx: 37},
			{price: 3, idx: 12},
			{price: 2, idx: 10},
		}, 1),
	}

	idx, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, 67, 5)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{5, 67},
		{4, 37},
		{3, 12},
		{2, 10},
	}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(10), best)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 7, idx)

	s.insertAt(idx, 81, 1)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{5, 67},
		{4, 37},
		{3, 12},
		{2, 10},
		{1, 81},
	}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(81), best)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, 22, 9)
	require.Equal(t, []entry{
		{9, 22},
		{8, 18},
		{7, 45},
		{6, 99},
		{5, 67},
		{4, 37},
		{3, 12},
		{2, 10},
		{1, 81},
	}, s.levels)
}

func TestBidRemove(t *testing.T) {
	s := side{
		bid: true,
		levels: []entry{
			{price: 2, idx: 10},
			{price: 3, idx: 12},
			{price: 4, idx: 37},
			{price: 6, idx: 99},
			{price: 7, idx: 45},
			{price: 8, idx: 18},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{2, 10},
		{3, 12},
		{6, 99},
		{7, 45},
		{8, 18},
	}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(18), best)

	idx, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{3, 12},
		{6, 99},
		{7, 45},
		{8, 18},
	}, s.levels)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{3, 12},
		{6, 99},
		{7, 45},
	}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(45), best)

}

func TestAskRemove(t *testing.T) {
	s := side{
		bid: false,
		levels: []entry{
			{price: 8, idx: 18},
			{price: 7, idx: 45},
			{price: 6, idx: 99},
			{price: 4, idx: 37},
			{price: 3, idx: 12},
			{price: 2, idx: 10},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{3, 12},
		{2, 10},
	}, s.levels)

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(10), best)

	idx, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, 4, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{3, 12},
	}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(12), best)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{7, 45},
		{6, 99},
		{3, 12},
	}, s.levels)
}

func TestEmpty(t *testing.T) {
	s := side{
		bid:    true,
		levels: []entry{},
	}

	best, found := s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)

	s.insertAt(0, 45, 7)
	require.Equal(t, []entry{{7, 45}}, s.levels)

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(45), best)

	s.removeAt(0)
	require.Equal(t, []entry{}, s.levels)

	best, found = s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)
}
