package book

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBidFind(t *testing.T) {
	s := side{
		bid: true,
		levels: []level{
			{price: 2},
			{price: 3},
			{price: 4},
			{price: 6},
			{price: 7},
			{price: 8},
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
		levels: []level{
			{price: 8},
			{price: 7},
			{price: 6},
			{price: 4},
			{price: 3},
			{price: 2},
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

func checkSide(t *testing.T, s side) {
	t.Helper()
	for i := 1; i < len(s.levels); i++ {
		if s.bid {
			require.Greater(t, s.levels[i].price, s.levels[i-1].price)
		} else {
			require.Less(t, s.levels[i].price, s.levels[i-1].price)
		}
	}
}

func TestBidInsert(t *testing.T) {
	s := side{
		bid: true,
		levels: []level{
			{price: 2},
			{price: 3},
			{price: 4},
			{price: 6},
			{price: 7},
			{price: 8},
		},
	}

	idx, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, 5, 0)
	require.Len(t, s.levels, 7)
	checkSide(t, s)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, 1, 0)
	require.Len(t, s.levels, 8)
	checkSide(t, s)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 8, idx)

	s.insertAt(idx, 9, 0)
	require.Len(t, s.levels, 9)
	checkSide(t, s)
}

func TestAskInsert(t *testing.T) {
	s := side{
		bid: false,
		levels: []level{
			{price: 8},
			{price: 7},
			{price: 6},
			{price: 4},
			{price: 3},
			{price: 2},
		},
	}

	idx, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, 3, idx)

	s.insertAt(idx, 5, 0)
	require.Len(t, s.levels, 7)
	checkSide(t, s)

	idx, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, 7, idx)

	s.insertAt(idx, 1, 0)
	require.Len(t, s.levels, 8)
	checkSide(t, s)

	idx, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, 0, idx)

	s.insertAt(idx, 9, 0)
	require.Len(t, s.levels, 9)
	checkSide(t, s)
}

func TestBidRemove(t *testing.T) {
	s := side{
		bid: true,
		levels: []level{
			{price: 2},
			{price: 3},
			{price: 4},
			{price: 6},
			{price: 7},
			{price: 8},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 2, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 5)
	checkSide(t, s)

	idx, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 4)
	checkSide(t, s)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 3)
	checkSide(t, s)
}

func TestAskRemove(t *testing.T) {
	s := side{
		bid: false,
		levels: []level{
			{price: 8},
			{price: 7},
			{price: 6},
			{price: 4},
			{price: 3},
			{price: 2},
		},
	}

	idx, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, 3, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 5)
	checkSide(t, s)

	idx, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, 4, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 4)
	checkSide(t, s)

	idx, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, 0, idx)

	s.removeAt(idx)
	require.Len(t, s.levels, 3)
	checkSide(t, s)
}
