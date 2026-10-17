package book

import (
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/levels"
)

func TestBook(t *testing.T) {
	book := New(10, 10)
	h1, err := book.Add(Buy, 100, 10)
	require.NoError(t, err)
	h2, err := book.Add(Buy, 100, 10)
	require.NoError(t, err)
	h3, err := book.Add(Sell, 101, 5)
	require.NoError(t, err)

	price, totalQty, ok := book.Best(Buy)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(20), totalQty)

	price, totalQty, ok = book.Best(Sell)
	require.True(t, ok)
	require.Equal(t, uint64(101), price)
	require.Equal(t, uint64(5), totalQty)

	require.NoError(t, book.Cancel(h1))
	price, totalQty, ok = book.Best(Buy)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(10), totalQty)

	require.NoError(t, book.Cancel(h2))
	_, _, ok = book.Best(Buy)
	require.False(t, ok)

	require.NoError(t, book.Cancel(h3))
	_, _, ok = book.Best(Sell)
	require.False(t, ok)

	require.ErrorIs(t, book.Cancel(h3), levels.ErrNotFound)

	h4, err := book.Add(Buy, 100, 0)
	require.ErrorIs(t, err, levels.ErrZeroQty)
	require.Equal(t, levels.NoHandle, h4)

	_, _, ok = book.Best(Buy)
	require.False(t, ok)

	h5, err := book.Add(Buy, 100, 1)
	require.NoError(t, err)
	_, err = book.Add(Sell, 100, 1)
	require.NoError(t, err)

	require.NoError(t, book.Cancel(h5))
	_, _, ok = book.Best(Buy)
	require.False(t, ok)
	price, totalQty, ok = book.Best(Sell)
	require.True(t, ok)
	require.Equal(t, uint64(100), price)
	require.Equal(t, uint64(1), totalQty)

	require.PanicsWithValue(t, "book: invalid side", func() { _, _ = book.Add(3, 1, 1) })
}
