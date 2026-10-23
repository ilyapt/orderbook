package book

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/levels"
)

// Reference model: one sorted slice of entries, best price at the end,
// exactly what side must look like when flattened.

func modelFind(m []entry, bid bool, price uint64) (int, bool) {
	for i, e := range m {
		if e.price == price {
			return i, true
		}
		if (bid && e.price > price) || (!bid && e.price < price) {
			return i, false
		}
	}
	return len(m), false
}

// compareSide checks the side against the model and its own invariants:
// flattened entries, chain keys, block fill, block arena size, best and
// atDepth at every depth.
func compareSide(t *testing.T, s *side, m []entry) {
	t.Helper()

	require.Equal(t, m, flat(s))
	require.Len(t, s.blocks, len(s.chain), "block arena vs chain")

	seen := map[int32]bool{}
	for c, ce := range s.chain {
		require.False(t, seen[ce.idx], "block %d linked twice", ce.idx)
		seen[ce.idx] = true
		b := &s.blocks[ce.idx]
		require.Greater(t, b.n, int32(0), "chain %d: empty block", c)
		require.LessOrEqual(t, b.n, int32(blockCap), "chain %d: overfull block", c)
		require.Equal(t, b.entries[0].price, ce.price, "chain %d: key", c)
	}

	best, ok := s.best()
	require.Equal(t, len(m) > 0, ok)
	if ok {
		require.Equal(t, levels.LevelIndex(m[len(m)-1].idx), best)
	}
	for d := range m {
		l, ok := s.atDepth(d)
		require.True(t, ok, "depth %d", d)
		require.Equal(t, levels.LevelIndex(m[len(m)-1-d].idx), l, "depth %d", d)
	}
	_, ok = s.atDepth(len(m))
	require.False(t, ok)

	cu, ok := s.cursor()
	require.Equal(t, len(m) > 0, ok)
	for d := len(m) - 1; d >= 0; d-- {
		require.True(t, ok, "cursor at %d", d)
		require.Equal(t, levels.LevelIndex(m[d].idx), cu.idx(), "cursor at %d", d)
		ok = cu.prev()
	}
	require.False(t, ok)

	out := make([]levels.LevelIndex, len(m)+1)
	require.Equal(t, len(m), s.depth(out))
	for d := range m {
		require.Equal(t, levels.LevelIndex(m[len(m)-1-d].idx), out[d], "depth %d", d)
	}
	if len(m) > 3 {
		require.Equal(t, 3, s.depth(out[:3]))
		require.Equal(t, levels.LevelIndex(m[len(m)-3].idx), out[2])
	}
}

func TestSideRandomAgainstModel(t *testing.T) {
	for _, bid := range []bool{true, false} {
		for seed := int64(1); seed <= 3; seed++ {
			t.Run(fmt.Sprintf("bid=%v/seed=%d", bid, seed), func(t *testing.T) {
				runSideRandom(t, bid, seed)
			})
		}
	}
}

func runSideRandom(t *testing.T, bid bool, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	s := side{bid: bid}
	var m []entry
	var nextIdx int32

	// Three phases: grow to several blocks, churn at a stable size,
	// shrink back to empty. Insert probability per phase.
	phases := []struct{ steps, insertPct int }{{6000, 85}, {6000, 50}, {6000, 15}}
	step := 0
	for _, ph := range phases {
		for k := 0; k < ph.steps; k++ {
			step++
			if len(m) == 0 || rng.Intn(100) < ph.insertPct {
				price := uint64(rng.Intn(10000) + 1)
				mi, found := modelFind(m, bid, price)
				p, ok := s.find(price)
				require.Equal(t, found, ok, "step %d: find %d", step, price)
				if found {
					require.Equal(t, levels.LevelIndex(m[mi].idx), s.at(p), "step %d: find %d", step, price)
					continue
				}
				nextIdx++
				e := entry{price: price, idx: nextIdx}
				s.insertAt(p, levels.LevelIndex(nextIdx), price)
				m = append(m[:mi], append([]entry{e}, m[mi:]...)...)
			} else {
				mi := rng.Intn(len(m))
				p, ok := s.find(m[mi].price)
				require.True(t, ok, "step %d: remove %d", step, m[mi].price)
				require.Equal(t, levels.LevelIndex(m[mi].idx), s.at(p), "step %d: remove %d", step, m[mi].price)
				s.removeAt(p)
				m = append(m[:mi], m[mi+1:]...)
			}

			if step%200 == 0 {
				compareSide(t, &s, m)
			}
		}
	}
	for len(m) > 0 {
		p, ok := s.find(m[0].price)
		require.True(t, ok)
		s.removeAt(p)
		m = m[1:]
	}
	compareSide(t, &s, m)
}
