package levels

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// Reference model: a level is a slice of orders in push order.
// No indexes or links; removal is a slice cut.

type modelOrder struct {
	h   Handle
	qty uint64
}

type modelLevel struct {
	price  uint64
	orders []modelOrder
}

func (m *modelLevel) total() uint64 {
	var sum uint64
	for _, o := range m.orders {
		sum += o.qty
	}
	return sum
}

func (m *modelLevel) remove(i int) {
	m.orders = append(m.orders[:i], m.orders[i+1:]...)
}

// compareModel checks every live level of Levels against the model:
// price, totalQty, order sequence, and each order's remaining qty and seq.
func compareModel(t *testing.T, l *Levels, model map[LevelIndex]*modelLevel) {
	t.Helper()

	live := 0
	for i := range l.levels {
		if l.levels[i].tail != -2 {
			live++
		}
	}
	require.Equal(t, len(model), live, "live level count")

	for id, m := range model {
		lvl := &l.levels[id]
		require.NotEqual(t, int32(-2), lvl.tail, "level %d is freed in Levels but live in model", id)
		require.Equal(t, m.price, lvl.price, "level %d price", id)
		require.Equal(t, m.total(), lvl.totalQty, "level %d totalQty", id)

		idx := make([]int32, len(m.orders))
		for i, o := range m.orders {
			idx[i] = o.h.index()
			got := &l.orders[o.h.index()]
			require.Equal(t, o.qty, got.qty, "level %d order %d qty", id, i)
			require.Equal(t, o.h.seq(), got.seq, "level %d order %d seq", id, i)
			require.Equal(t, id, got.level, "level %d order %d owner", id, i)
		}
		chainEqual(t, l, lvl, idx...)
	}
}

func TestRandomAgainstModel(t *testing.T) {
	const steps = 20000
	for seed := int64(1); seed <= 5; seed++ {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			runRandom(t, seed, steps)
		})
	}
}

func runRandom(t *testing.T, seed int64, steps int) {
	rng := rand.New(rand.NewSource(seed))

	// Small initial capacities so that append and slice growth happen often.
	l := New(4, 2)
	model := map[LevelIndex]*modelLevel{}
	var liveLevels []LevelIndex // keys of model, for picking a random level
	var stale []Handle          // handles of cancelled and fully filled orders

	pickLevel := func() LevelIndex {
		return liveLevels[rng.Intn(len(liveLevels))]
	}
	dropLevel := func(id LevelIndex) {
		for i, x := range liveLevels {
			if x == id {
				liveLevels[i] = liveLevels[len(liveLevels)-1]
				liveLevels = liveLevels[:len(liveLevels)-1]
				return
			}
		}
		t.Fatalf("level %d not in liveLevels", id)
	}
	// A random live order: its level and position in the queue.
	pickOrder := func() (LevelIndex, int, bool) {
		var cands []LevelIndex
		for _, id := range liveLevels {
			if len(model[id].orders) > 0 {
				cands = append(cands, id)
			}
		}
		if len(cands) == 0 {
			return 0, 0, false
		}
		id := cands[rng.Intn(len(cands))]
		return id, rng.Intn(len(model[id].orders)), true
	}

	for step := 0; step < steps; step++ {
		switch op := rng.Intn(100); {

		case op < 10: // new level
			price := uint64(rng.Intn(1000) + 1)
			id := l.NewLevel(price)
			_, exists := model[id]
			require.False(t, exists, "step %d: NewLevel returned live index %d", step, id)
			model[id] = &modelLevel{price: price}
			liveLevels = append(liveLevels, id)

		case op < 50: // push
			if len(liveLevels) == 0 {
				continue
			}
			id := pickLevel()
			qty := uint64(rng.Intn(100) + 1)
			h, err := l.Push(id, qty)
			require.NoError(t, err, "step %d", step)
			model[id].orders = append(model[id].orders, modelOrder{h: h, qty: qty})

		case op < 70: // cancel a live order
			id, i, ok := pickOrder()
			if !ok {
				continue
			}
			m := model[id]
			o := m.orders[i]
			gotID, empty, err := l.Cancel(o.h)
			require.NoError(t, err, "step %d", step)
			require.Equal(t, id, gotID, "step %d", step)
			m.remove(i)
			require.Equal(t, len(m.orders) == 0, empty, "step %d: empty flag", step)
			stale = append(stale, o.h)

		case op < 85: // fill, partial or full
			id, i, ok := pickOrder()
			if !ok {
				continue
			}
			m := model[id]
			o := m.orders[i]
			fill := uint64(rng.Int63n(int64(o.qty))) + 1 // 1..qty
			gotID, empty := l.Fill(o.h, fill)
			require.Equal(t, id, gotID, "step %d", step)
			if fill == o.qty {
				m.remove(i)
				stale = append(stale, o.h)
			} else {
				m.orders[i].qty -= fill
			}
			require.Equal(t, len(m.orders) == 0, empty, "step %d: empty flag", step)

		case op < 92: // cancel by a stale handle
			if len(stale) == 0 {
				continue
			}
			h := stale[rng.Intn(len(stale))]
			_, _, err := l.Cancel(h)
			require.ErrorIs(t, err, ErrNotFound, "step %d", step)

		case op < 95: // zero qty leaves state untouched
			if len(liveLevels) == 0 {
				continue
			}
			seq, n := l.seq, len(l.orders)
			h, err := l.Push(pickLevel(), 0)
			require.ErrorIs(t, err, ErrZeroQty, "step %d", step)
			require.Equal(t, NoHandle, h)
			require.Equal(t, seq, l.seq, "step %d: seq consumed on error", step)
			require.Len(t, l.orders, n, "step %d: slot allocated on error", step)

		default: // free an empty level
			var empties []LevelIndex
			for _, id := range liveLevels {
				if len(model[id].orders) == 0 {
					empties = append(empties, id)
				}
			}
			if len(empties) == 0 {
				continue
			}
			id := empties[rng.Intn(len(empties))]
			l.FreeLevel(id)
			delete(model, id)
			dropLevel(id)
		}

		checkInvariants(t, l)
		compareModel(t, l, model)
	}
}
