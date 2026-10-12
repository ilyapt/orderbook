package levels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func checkInvariants(t *testing.T, l *Levels) {
	t.Helper()

	levelsLen := len(l.levels)
	ordersLen := len(l.orders)

	for i, lvl := range l.levels {
		if lvl.tail == -2 {
			continue
		}

		if lvl.head == -1 || lvl.tail == -1 || lvl.totalQty == 0 {
			require.True(t, lvl.head == -1 && lvl.tail == -1 && lvl.totalQty == 0)
		}

		p, n, totalQty := int32(-1), lvl.head, lvl.totalQty
		for n != -1 {
			o := &l.orders[n]
			require.Equal(t, p, o.prev)
			require.LessOrEqual(t, o.qty, totalQty)
			require.Equal(t, LevelIndex(i), o.level)
			require.Greater(t, ordersLen, 0)
			totalQty -= o.qty
			p = n
			n = o.next
			ordersLen--
		}
		require.Equal(t, uint64(0), totalQty)

		if lvl.head != -1 {
			require.Equal(t, int32(-1), l.orders[lvl.head].prev)
		}
		if lvl.tail != -1 {
			require.Equal(t, int32(-1), l.orders[lvl.tail].next)
		}

		levelsLen--
	}

	n := l.freeOrder
	for n != -1 {
		require.Equal(t, LevelIndex(-1), l.orders[n].level)
		require.Equal(t, int32(-1), l.orders[n].prev)
		require.Greater(t, ordersLen, 0)
		n = l.orders[n].next
		ordersLen--
	}
	require.Equal(t, 0, ordersLen)

	n = l.freeLevel
	for n != -1 {
		require.Equal(t, int32(-2), l.levels[n].tail)
		require.Greater(t, levelsLen, 0)
		n = l.levels[n].head
		levelsLen--
	}
	require.Equal(t, 0, levelsLen)
}

func chainEqual(t *testing.T, l *Levels, lvl *level, orderIndexes ...int32) {
	t.Helper()

	i, n := 0, lvl.head
	for n != -1 {
		require.Less(t, i, len(orderIndexes))
		require.Equal(t, orderIndexes[i], n)
		n = l.orders[n].next
		i++
	}
	require.Equal(t, len(orderIndexes), i)
}

func TestOrderLifecycle(t *testing.T) {
	l := New(10, 10)
	lvlIdx := l.NewLevel(5)
	oh, err := l.Push(lvlIdx, 100)
	require.NoError(t, err)
	checkInvariants(t, l)
	require.Equal(t, uint64(1), oh.seq())
	require.Equal(t, int32(0), oh.index())

	li, empty := l.Fill(oh, 50)
	require.False(t, empty)
	require.Equal(t, lvlIdx, li)
	require.Equal(t, uint64(50), l.levels[lvlIdx].totalQty)
	checkInvariants(t, l)

	li, empty = l.Fill(oh, 50)
	require.True(t, empty)
	require.Equal(t, lvlIdx, li)
	require.Equal(t, uint64(0), l.levels[lvlIdx].totalQty)
	checkInvariants(t, l)

	li, empty, err = l.Cancel(oh)
	require.ErrorIs(t, err, ErrNotFound)

	oh2, err := l.Push(lvlIdx, 10)
	require.NoError(t, err)
	require.Equal(t, oh.index(), oh2.index())
	require.Equal(t, uint64(2), oh2.seq())

	checkInvariants(t, l)

	li, empty, err = l.Cancel(oh)
	require.ErrorIs(t, err, ErrNotFound)

	li, empty, err = l.Cancel(oh2)
	require.NoError(t, err)
	require.True(t, empty)
	require.Equal(t, lvlIdx, li)

	checkInvariants(t, l)
}

func TestQueueOrderAfterRemovals(t *testing.T) {
	l := New(10, 10)
	lvlIdx := l.NewLevel(5)

	oh1, err := l.Push(lvlIdx, 100)
	require.NoError(t, err)
	checkInvariants(t, l)

	oh2, err := l.Push(lvlIdx, 50)
	require.NoError(t, err)
	checkInvariants(t, l)

	oh3, err := l.Push(lvlIdx, 200)
	require.NoError(t, err)
	checkInvariants(t, l)

	chainEqual(t, l, &l.levels[lvlIdx], oh1.index(), oh2.index(), oh3.index())

	li, empty, err := l.Cancel(oh2)
	require.NoError(t, err)
	require.False(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)

	chainEqual(t, l, &l.levels[lvlIdx], oh1.index(), oh3.index())

	li, empty, err = l.Cancel(oh1)
	require.NoError(t, err)
	require.False(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)

	chainEqual(t, l, &l.levels[lvlIdx], oh3.index())

	li, empty, err = l.Cancel(oh3)
	require.NoError(t, err)
	require.True(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)
}
