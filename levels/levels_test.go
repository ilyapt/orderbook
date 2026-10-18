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
	require.Equal(t, uint64(1), oh.Seq())
	require.Equal(t, int32(0), oh.index())

	li, q, empty := l.Fill(oh, 50)
	require.False(t, empty)
	require.Equal(t, uint64(50), q)
	require.Equal(t, lvlIdx, li)
	require.Equal(t, uint64(50), l.levels[lvlIdx].totalQty)
	checkInvariants(t, l)

	require.PanicsWithValue(t, "levels: Fill: qty exceeds order remainder", func() { l.Fill(oh, 150) })

	li, q, empty = l.Fill(oh, 50)
	require.True(t, empty)
	require.Equal(t, uint64(0), q)
	require.Equal(t, lvlIdx, li)
	require.Equal(t, uint64(0), l.levels[lvlIdx].totalQty)
	checkInvariants(t, l)

	_, _, _, err = l.Cancel(oh)
	require.ErrorIs(t, err, ErrNotFound)

	oh2, err := l.Push(lvlIdx, 10)
	require.NoError(t, err)
	require.Equal(t, oh.index(), oh2.index())
	require.Equal(t, uint64(2), oh2.Seq())

	checkInvariants(t, l)

	ln := len(l.orders)
	ls := l.seq
	oh3, err := l.Push(lvlIdx, 0)
	require.ErrorIs(t, err, ErrZeroQty)
	require.Equal(t, NoHandle, oh3)
	require.Equal(t, ls, l.seq)
	require.Len(t, l.orders, ln)

	_, _, _, err = l.Cancel(oh)
	require.ErrorIs(t, err, ErrNotFound)

	require.PanicsWithValue(t, "levels: Fill: stale handle", func() { l.Fill(oh, 100) })

	li, remains, empty, err := l.Cancel(oh2)
	require.NoError(t, err)
	require.Equal(t, uint64(10), remains)
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

	li, remains, empty, err := l.Cancel(oh2)
	require.NoError(t, err)
	require.Equal(t, uint64(50), remains)
	require.False(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)

	chainEqual(t, l, &l.levels[lvlIdx], oh1.index(), oh3.index())

	li, remains, empty, err = l.Cancel(oh1)
	require.NoError(t, err)
	require.Equal(t, uint64(100), remains)
	require.False(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)

	chainEqual(t, l, &l.levels[lvlIdx], oh3.index())

	li, remains, empty, err = l.Cancel(oh3)
	require.NoError(t, err)
	require.Equal(t, uint64(200), remains)
	require.True(t, empty)
	require.Equal(t, lvlIdx, li)
	checkInvariants(t, l)
}

func TestLevelsReuse(t *testing.T) {
	l := New(10, 10)
	lvl1 := l.NewLevel(100)
	lvl2 := l.NewLevel(200)

	o1, err := l.Push(lvl1, 10)
	require.NoError(t, err)
	o2, err := l.Push(lvl2, 20)
	require.NoError(t, err)
	checkInvariants(t, l)

	li, remains, empty, err := l.Cancel(o1)
	require.NoError(t, err)
	require.Equal(t, uint64(10), remains)
	require.True(t, empty)
	require.Equal(t, lvl1, li)
	checkInvariants(t, l)

	l.FreeLevel(lvl1)
	require.Equal(t, int32(-2), l.levels[lvl1].tail)
	require.Equal(t, int32(lvl1), l.freeLevel)
	checkInvariants(t, l)

	require.PanicsWithValue(t, "levels: FreeLevel: level already freed", func() { l.FreeLevel(lvl1) })

	require.PanicsWithValue(t, "levels: FreeLevel: level is not empty", func() { l.FreeLevel(lvl2) })

	require.PanicsWithValue(t, "levels: level is freed", func() { _, _ = l.Push(lvl1, 100) })

	li, remains, empty, err = l.Cancel(o2)
	require.NoError(t, err)
	require.Equal(t, uint64(20), remains)
	require.True(t, empty)
	require.Equal(t, lvl2, li)
	checkInvariants(t, l)

	l.FreeLevel(lvl2)
	require.Equal(t, int32(-2), l.levels[lvl2].tail)
	require.Equal(t, int32(lvl1), l.levels[lvl2].head)
	require.Equal(t, int32(lvl2), l.freeLevel)
	checkInvariants(t, l)

	lvl3 := l.NewLevel(300)
	require.Equal(t, lvl2, lvl3)
	require.Equal(t, uint64(300), l.levels[lvl3].price)
	require.Equal(t, uint64(0), l.levels[lvl3].totalQty)
	require.Equal(t, int32(-1), l.levels[lvl3].head)
	require.Equal(t, int32(-1), l.levels[lvl3].tail)
	require.Equal(t, int32(lvl1), l.freeLevel)
	checkInvariants(t, l)

	o3, err := l.Push(lvl3, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), l.levels[lvl3].totalQty)
	require.Equal(t, o3.index(), l.levels[lvl3].head)
	require.Equal(t, o3.index(), l.levels[lvl3].tail)
	checkInvariants(t, l)

	lvl4 := l.NewLevel(400)
	require.Equal(t, lvl1, lvl4)
	require.Equal(t, uint64(400), l.levels[lvl4].price)
	require.Equal(t, uint64(0), l.levels[lvl4].totalQty)
	require.Equal(t, int32(-1), l.levels[lvl4].head)
	require.Equal(t, int32(-1), l.levels[lvl4].tail)
	require.Equal(t, int32(-1), l.freeLevel)
	require.Len(t, l.levels, 2)
	checkInvariants(t, l)

	lvl5 := l.NewLevel(500)
	require.Equal(t, LevelIndex(2), lvl5)
	require.Equal(t, uint64(500), l.levels[lvl5].price)
	require.Equal(t, uint64(0), l.levels[lvl5].totalQty)
	require.Equal(t, int32(-1), l.levels[lvl5].head)
	require.Equal(t, int32(-1), l.levels[lvl5].tail)
	require.Equal(t, int32(-1), l.freeLevel)
	require.Len(t, l.levels, 3)
	checkInvariants(t, l)
}

func TestHandlePacking(t *testing.T) {
	require.Equal(t, Handle(1<<indexBits), makeHandle(0, 1))
	require.Equal(t, Handle(1<<indexBits|indexMask), makeHandle(indexMask, 1))
	require.Equal(t, Handle(seqMax<<indexBits), makeHandle(0, seqMax))
	require.Equal(t, Handle(seqMax<<indexBits|indexMask), makeHandle(indexMask, seqMax))

	h := Handle(seqMax<<indexBits | indexMask)
	require.Equal(t, int32(indexMask), h.index())
	require.Equal(t, uint64(seqMax), h.Seq())

	h = Handle(1<<indexBits | indexMask)
	require.Equal(t, int32(indexMask), h.index())
	require.Equal(t, uint64(1), h.Seq())

	h = Handle(seqMax << indexBits)
	require.Equal(t, int32(0), h.index())
	require.Equal(t, uint64(seqMax), h.Seq())

	require.Equal(t, int32(0), NoHandle.index())
	require.Equal(t, uint64(0), NoHandle.Seq())
}
