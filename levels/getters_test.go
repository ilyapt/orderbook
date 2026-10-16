package levels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetters(t *testing.T) {
	l := New(10, 10)
	lvl := l.NewLevel(7)
	require.Equal(t, uint64(7), l.Price(lvl))
	require.Equal(t, uint64(0), l.TotalQty(lvl))

	h, ok := l.Front(lvl)
	require.False(t, ok)
	require.Equal(t, NoHandle, h)

	o1, err := l.Push(lvl, 100)
	require.NoError(t, err)
	o2, err := l.Push(lvl, 50)
	require.NoError(t, err)
	require.Equal(t, uint64(150), l.TotalQty(lvl))

	h, ok = l.Front(lvl)
	require.True(t, ok)
	require.Equal(t, o1, h)

	q, err := l.Qty(o1)
	require.NoError(t, err)
	require.Equal(t, uint64(100), q)

	li, err := l.Level(o2)
	require.NoError(t, err)
	require.Equal(t, lvl, li)

	li, empty := l.Fill(o1, 30)
	require.False(t, empty)
	require.Equal(t, lvl, li)
	q, err = l.Qty(o1)
	require.NoError(t, err)
	require.Equal(t, uint64(70), q)
	require.Equal(t, uint64(120), l.TotalQty(lvl))

	li, empty, err = l.Cancel(o1)
	require.NoError(t, err)
	require.False(t, empty)
	require.Equal(t, lvl, li)

	h, ok = l.Front(lvl)
	require.True(t, ok)
	require.Equal(t, o2, h)

	q, err = l.Qty(o1)
	require.ErrorIs(t, err, ErrNotFound)
	require.Equal(t, uint64(0), q)
	li, err = l.Level(o1)
	require.ErrorIs(t, err, ErrNotFound)
	require.Equal(t, LevelIndex(0), li)

	_, empty, err = l.Cancel(o2)
	require.NoError(t, err)
	require.True(t, empty)
	require.Equal(t, uint64(0), l.TotalQty(lvl))

	h, ok = l.Front(lvl)
	require.False(t, ok)
	require.Equal(t, NoHandle, h)

	l.FreeLevel(lvl)
	require.PanicsWithValue(t, "levels: level is freed", func() { l.Price(lvl) })
	require.PanicsWithValue(t, "levels: level is freed", func() { l.TotalQty(lvl) })
	require.PanicsWithValue(t, "levels: level is freed", func() { l.Front(lvl) })
	require.PanicsWithValue(t, "levels: level index out of range", func() { l.Price(LevelIndex(-1)) })
	require.PanicsWithValue(t, "levels: level index out of range", func() { l.Price(LevelIndex(len(l.levels))) })
}
