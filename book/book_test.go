package book

import (
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/levels"
)

func TestBook(t *testing.T) {
	book := New(10, 10)
	h1, err := book.Add(Buy, 1, 100, 10)
	require.NoError(t, err)
	h2, err := book.Add(Buy, 2, 100, 10)
	require.NoError(t, err)
	h3, err := book.Add(Sell, 3, 101, 5)
	require.NoError(t, err)

	price, totalQty, ok := book.Best(Buy)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(20), totalQty)

	price, totalQty, ok = book.Best(Sell)
	require.True(t, ok)
	require.Equal(t, uint64(101), price)
	require.Equal(t, uint64(5), totalQty)

	remaining, err := book.Cancel(h1)
	require.NoError(t, err)
	require.Equal(t, uint64(10), remaining)
	price, totalQty, ok = book.Best(Buy)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(10), totalQty)

	remaining, err = book.Cancel(h2)
	require.NoError(t, err)
	require.Equal(t, uint64(10), remaining)
	_, _, ok = book.Best(Buy)
	require.False(t, ok)

	remaining, err = book.Cancel(h3)
	require.NoError(t, err)
	require.Equal(t, uint64(5), remaining)
	_, _, ok = book.Best(Sell)
	require.False(t, ok)

	_, err = book.Cancel(h3)
	require.ErrorIs(t, err, levels.ErrNotFound)

	h4, err := book.Add(Buy, 4, 100, 0)
	require.ErrorIs(t, err, levels.ErrZeroQty)
	require.Equal(t, levels.NoHandle, h4)

	_, _, ok = book.Best(Buy)
	require.False(t, ok)

	h5, err := book.Add(Buy, 5, 100, 1)
	require.NoError(t, err)
	_, err = book.Add(Sell, 6, 100, 1)
	require.NoError(t, err)

	remaining, err = book.Cancel(h5)
	require.Equal(t, uint64(1), remaining)
	require.NoError(t, err)
	_, _, ok = book.Best(Buy)
	require.False(t, ok)
	price, totalQty, ok = book.Best(Sell)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(1), totalQty)

	require.PanicsWithValue(t, "book: invalid side", func() { _, _ = book.Add(3, 7, 1, 1) })
}

func TestFrontFill(t *testing.T) {
	book := New(10, 10)
	h1, err := book.Add(Buy, 1, 100, 10)
	require.NoError(t, err)
	h2, err := book.Add(Buy, 2, 100, 5)
	require.NoError(t, err)
	h3, err := book.Add(Sell, 3, 101, 7)
	require.NoError(t, err)

	h, price, qty, ok := book.Front(Buy)
	require.True(t, ok)
	require.Equal(t, h1, h)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(10), qty)

	require.Equal(t, uint64(6), book.Fill(h1, 4))
	h, _, qty, ok = book.Front(Buy)
	require.True(t, ok)
	require.Equal(t, h1, h)
	require.Equal(t, uint64(6), qty)
	_, totalQty, _ := book.Best(Buy)
	require.Equal(t, uint64(11), totalQty)

	require.Equal(t, uint64(0), book.Fill(h1, 6))
	h, _, qty, ok = book.Front(Buy)
	require.True(t, ok)
	require.Equal(t, h2, h)
	require.Equal(t, uint64(5), qty)

	require.Equal(t, uint64(0), book.Fill(h2, 5))
	_, _, _, ok = book.Front(Buy)
	require.False(t, ok)
	_, _, ok = book.Best(Buy)
	require.False(t, ok)

	h, price, qty, ok = book.Front(Sell)
	require.True(t, ok)
	require.Equal(t, h3, h)
	require.Equal(t, uint64(101), price)
	require.Equal(t, uint64(7), qty)

	require.PanicsWithValue(t, "levels: Fill: stale handle", func() { book.Fill(h1, 1) })
}

func TestLevelAt(t *testing.T) {
	book := New(10, 10)
	_, _, ok := book.LevelAt(Buy, 0)
	require.False(t, ok)

	_, err := book.Add(Buy, 1, 100, 10)
	require.NoError(t, err)
	_, err = book.Add(Buy, 2, 98, 3)
	require.NoError(t, err)
	_, err = book.Add(Buy, 3, 99, 2)
	require.NoError(t, err)
	_, err = book.Add(Sell, 4, 101, 7)
	require.NoError(t, err)
	_, err = book.Add(Sell, 5, 103, 1)
	require.NoError(t, err)

	price, totalQty, ok := book.LevelAt(Buy, 0)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(10), totalQty)

	price, totalQty, ok = book.LevelAt(Buy, 1)
	require.True(t, ok)
	require.Equal(t, uint64(99), price)
	require.Equal(t, uint64(2), totalQty)

	price, totalQty, ok = book.LevelAt(Buy, 2)
	require.True(t, ok)
	require.Equal(t, uint64(98), price)
	require.Equal(t, uint64(3), totalQty)

	_, _, ok = book.LevelAt(Buy, 3)
	require.False(t, ok)

	price, totalQty, ok = book.LevelAt(Sell, 0)
	require.True(t, ok)
	require.Equal(t, uint64(101), price)
	require.Equal(t, uint64(7), totalQty)

	price, totalQty, ok = book.LevelAt(Sell, 1)
	require.True(t, ok)
	require.Equal(t, uint64(103), price)
	require.Equal(t, uint64(1), totalQty)

	_, _, ok = book.LevelAt(Sell, 2)
	require.False(t, ok)
}
