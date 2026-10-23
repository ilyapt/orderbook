package book

import (
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/levels"
)

func newSide(bid bool, entries []entry) side {
	s := side{bid: bid}
	for _, e := range entries {
		p, _, _ := s.find(e.price)
		s.insertAt(p, levels.LevelIndex(e.idx), e.price)
	}
	return s
}

func flat(s *side) []entry {
	out := []entry{}
	for _, c := range s.chain {
		b := &s.blocks[c.idx]
		out = append(out, b.entries[:b.n]...)
	}
	return out
}

func TestBidFind(t *testing.T) {
	s := newSide(true, []entry{
		{price: 2, idx: 10},
		{price: 3, idx: 12},
		{price: 4, idx: 37},
		{price: 6, idx: 99},
		{price: 7, idx: 45},
		{price: 8, idx: 18},
	})

	idx, _, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, pos{0, 2}, idx)

	idx, _, ok = s.find(5)
	require.False(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	idx, _, ok = s.find(6)
	require.True(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	idx, _, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, pos{0, 0}, idx)

	idx, _, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, pos{0, 5}, idx)

	idx, _, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, pos{0, 6}, idx)
}

func TestAskFind(t *testing.T) {
	s := newSide(false, []entry{
		{price: 8, idx: 18},
		{price: 7, idx: 45},
		{price: 6, idx: 99},
		{price: 4, idx: 37},
		{price: 3, idx: 12},
		{price: 2, idx: 10},
	})

	idx, _, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	idx, _, ok = s.find(5)
	require.False(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	idx, _, ok = s.find(6)
	require.True(t, ok)
	require.Equal(t, pos{0, 2}, idx)

	idx, _, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, pos{0, 6}, idx)

	idx, _, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, pos{0, 0}, idx)

	idx, _, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, pos{0, 0}, idx)
}

func TestBidInsert(t *testing.T) {
	s := newSide(true, []entry{
		{price: 2, idx: 10},
		{price: 3, idx: 12},
		{price: 4, idx: 37},
		{price: 6, idx: 99},
		{price: 7, idx: 45},
		{price: 8, idx: 18},
	})

	idx, _, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	s.insertAt(idx, 67, 5)
	require.Equal(t, []entry{
		{2, 10},
		{3, 12},
		{4, 37},
		{5, 67},
		{6, 99},
		{7, 45},
		{8, 18},
	}, flat(&s))

	idx, _, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, pos{0, 0}, idx)

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
	}, flat(&s))

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(18), best)

	idx, _, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, pos{0, 8}, idx)

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
	}, flat(&s))

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(22), best)
}

func TestAskInsert(t *testing.T) {
	s := newSide(false, []entry{
		{price: 8, idx: 18},
		{price: 7, idx: 45},
		{price: 6, idx: 99},
		{price: 4, idx: 37},
		{price: 3, idx: 12},
		{price: 2, idx: 10},
	})

	idx, _, ok := s.find(5)
	require.False(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	s.insertAt(idx, 67, 5)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{5, 67},
		{4, 37},
		{3, 12},
		{2, 10},
	}, flat(&s))

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(10), best)

	idx, _, ok = s.find(1)
	require.False(t, ok)
	require.Equal(t, pos{0, 7}, idx)

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
	}, flat(&s))

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(81), best)

	idx, _, ok = s.find(9)
	require.False(t, ok)
	require.Equal(t, pos{0, 0}, idx)

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
	}, flat(&s))
}

func TestBidRemove(t *testing.T) {
	s := newSide(true, []entry{
		{price: 2, idx: 10},
		{price: 3, idx: 12},
		{price: 4, idx: 37},
		{price: 6, idx: 99},
		{price: 7, idx: 45},
		{price: 8, idx: 18},
	})

	idx, _, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, pos{0, 2}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{2, 10},
		{3, 12},
		{6, 99},
		{7, 45},
		{8, 18},
	}, flat(&s))

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(18), best)

	idx, _, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, pos{0, 0}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{3, 12},
		{6, 99},
		{7, 45},
		{8, 18},
	}, flat(&s))

	idx, _, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{3, 12},
		{6, 99},
		{7, 45},
	}, flat(&s))

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(45), best)

}

func TestAskRemove(t *testing.T) {
	s := newSide(false, []entry{
		{price: 8, idx: 18},
		{price: 7, idx: 45},
		{price: 6, idx: 99},
		{price: 4, idx: 37},
		{price: 3, idx: 12},
		{price: 2, idx: 10},
	})

	idx, _, ok := s.find(4)
	require.True(t, ok)
	require.Equal(t, pos{0, 3}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{3, 12},
		{2, 10},
	}, flat(&s))

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(10), best)

	idx, _, ok = s.find(2)
	require.True(t, ok)
	require.Equal(t, pos{0, 4}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{8, 18},
		{7, 45},
		{6, 99},
		{3, 12},
	}, flat(&s))

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(12), best)

	idx, _, ok = s.find(8)
	require.True(t, ok)
	require.Equal(t, pos{0, 0}, idx)

	s.removeAt(idx)
	require.Equal(t, []entry{
		{7, 45},
		{6, 99},
		{3, 12},
	}, flat(&s))
}

func TestEmpty(t *testing.T) {
	s := newSide(true, nil)

	best, found := s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)

	s.insertAt(pos{0, 0}, 45, 7)
	require.Equal(t, []entry{{7, 45}}, flat(&s))

	best, found = s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(45), best)

	s.removeAt(pos{0, 0})
	require.Equal(t, []entry{}, flat(&s))

	best, found = s.best()
	require.False(t, found)
	require.Equal(t, levels.LevelIndex(-1), best)
}

func TestSplit(t *testing.T) {
	s := newSide(true, nil)
	for p := uint64(2); p <= 2*blockCap; p += 2 {
		i, _, ok := s.find(p)
		require.False(t, ok)
		s.insertAt(i, levels.LevelIndex(p), p)
	}
	require.Len(t, s.chain, 1)
	require.Len(t, s.blocks, 1)
	require.Equal(t, int32(blockCap), s.blocks[0].n)

	i, _, ok := s.find(2*blockCap + 2)
	require.False(t, ok)
	require.Equal(t, pos{0, blockCap}, i)
	s.insertAt(i, 2*blockCap+2, 2*blockCap+2)
	require.Len(t, s.chain, 2)
	require.Len(t, s.blocks, 2)
	require.Equal(t, entry{2, 0}, s.chain[0])
	require.Equal(t, entry{blockCap + 2, 1}, s.chain[1])
	require.Equal(t, int32(blockCap/2), s.blocks[0].n)
	require.Equal(t, int32(blockCap/2+1), s.blocks[1].n)

	i, _, ok = s.find(blockCap + 2)
	require.True(t, ok)
	require.Equal(t, pos{1, 0}, i)
	i, _, ok = s.find(blockCap)
	require.True(t, ok)
	require.Equal(t, pos{0, blockCap/2 - 1}, i)

	i, _, ok = s.find(blockCap + 1)
	require.False(t, ok)
	require.Equal(t, pos{0, blockCap / 2}, i)
	s.insertAt(i, blockCap+1, blockCap+1)
	require.Equal(t, int32(blockCap/2+1), s.blocks[0].n)
	require.Equal(t, entry{blockCap + 2, 1}, s.chain[1])

	all := flat(&s)
	require.Len(t, all, blockCap+2)
	for j := 1; j < len(all); j++ {
		require.Less(t, all[j-1].price, all[j].price)
	}

	best, found := s.best()
	require.True(t, found)
	require.Equal(t, levels.LevelIndex(2*blockCap+2), best)
	cu, ok := s.cursor()
	require.True(t, ok)
	require.Equal(t, levels.LevelIndex(2*blockCap+2), cu.idx())
	for d := 0; d < blockCap/2; d++ {
		require.True(t, cu.prev())
	}
	require.Equal(t, levels.LevelIndex(blockCap+2), cu.idx())
	require.True(t, cu.prev())
	require.Equal(t, levels.LevelIndex(blockCap+1), cu.idx())
	for d := blockCap/2 + 1; d < blockCap+1; d++ {
		require.True(t, cu.prev())
	}
	require.Equal(t, levels.LevelIndex(2), cu.idx())
	require.False(t, cu.prev())

	for len(s.chain) == 2 {
		s.removeAt(pos{0, 0})
	}
	require.Len(t, s.blocks, 1)
	require.Equal(t, entry{blockCap + 2, 0}, s.chain[0])
	require.Equal(t, int32(blockCap/2+1), s.blocks[0].n)
	all = flat(&s)
	require.Len(t, all, blockCap/2+1)
	require.Equal(t, uint64(blockCap+2), all[0].price)
	require.Equal(t, uint64(2*blockCap+2), all[len(all)-1].price)

	for len(s.chain) == 1 {
		s.removeAt(pos{0, 0})
	}
	require.Empty(t, s.blocks)
	require.Equal(t, []entry{}, flat(&s))
	_, found = s.best()
	require.False(t, found)
}
