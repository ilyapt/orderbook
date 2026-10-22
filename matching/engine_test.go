package matching

import (
	"testing"

	"github.com/stretchr/testify/require"

	"orderbook/book"
	"orderbook/levels"
)

type trades []Trade

func (s *trades) OnTrade(t Trade) { *s = append(*s, t) }

func TestPlace(t *testing.T) {
	var got trades
	e, _ := New(10, 10, &got)

	_, err := e.Place(Order{TIF: GTC, Side: book.Buy, Price: 100, Qty: 0})
	require.ErrorIs(t, err, ErrZeroQty)
	_, err = e.Place(Order{TIF: GTC, Side: book.Buy, Price: 0, Qty: 1})
	require.ErrorIs(t, err, ErrZeroPrice)
	_, err = e.Place(Order{TIF: 0, Side: book.Buy, Price: 100, Qty: 1})
	require.ErrorIs(t, err, ErrInvalidTIF)

	got = got[:0]
	r1, err := e.Place(Order{TIF: GTC, Side: book.Buy, Price: 100, Qty: 10})
	require.NoError(t, err)
	require.Equal(t, Open, r1.Status)
	require.Equal(t, uint64(10), r1.RemainingQty)
	require.Empty(t, got)
	require.Equal(t, uint64(1), levels.Handle(r1.OrderID).Seq())

	r2, err := e.Place(Order{TIF: GTC, Side: book.Buy, Price: 99, Qty: 5})
	require.NoError(t, err)
	require.Equal(t, Open, r2.Status)

	got = got[:0]
	r3, err := e.Place(Order{TIF: GTC, Side: book.Sell, Price: 100, Qty: 4})
	require.NoError(t, err)
	require.Equal(t, Filled, r3.Status)
	require.Equal(t, uint64(0), r3.RemainingQty)
	require.Equal(t, uint64(3), levels.Handle(r3.OrderID).Seq())
	require.Equal(t, trades{{ID: 1, TakerOrderID: r3.OrderID, MakerOrderID: r1.OrderID, Price: 100, Qty: 4, TakerRemaining: 0, MakerRemaining: 6}}, got)

	got = got[:0]
	r4, err := e.Place(Order{TIF: IOC, Side: book.Sell, Price: 99, Qty: 20})
	require.NoError(t, err)
	require.Equal(t, Canceled, r4.Status)
	require.Equal(t, uint64(9), r4.RemainingQty)
	require.Equal(t, uint64(4), levels.Handle(r4.OrderID).Seq())
	require.Equal(t, trades{
		{ID: 2, TakerOrderID: r4.OrderID, MakerOrderID: r1.OrderID, Price: 100, Qty: 6, TakerRemaining: 14, MakerRemaining: 0},
		{ID: 3, TakerOrderID: r4.OrderID, MakerOrderID: r2.OrderID, Price: 99, Qty: 5, TakerRemaining: 9, MakerRemaining: 0},
	}, got)
	_, _, ok := e.book.Best(book.Buy)
	require.False(t, ok)

	r5, err := e.Place(Order{TIF: GTC, Side: book.Sell, Price: 101, Qty: 3})
	require.NoError(t, err)
	require.Equal(t, Open, r5.Status)

	got = got[:0]
	r6, err := e.Place(Order{TIF: FOK, Side: book.Buy, Price: 101, Qty: 4})
	require.NoError(t, err)
	require.Equal(t, Rejected, r6.Status)
	require.Equal(t, uint64(4), r6.RemainingQty)
	require.Empty(t, got)
	_, totalQty, ok := e.book.Best(book.Sell)
	require.True(t, ok)
	require.Equal(t, uint64(3), totalQty)

	got = got[:0]
	r7, err := e.Place(Order{TIF: FOK, Side: book.Buy, Price: 101, Qty: 3})
	require.NoError(t, err)
	require.Equal(t, Filled, r7.Status)
	require.Len(t, got, 1)
	_, _, ok = e.book.Best(book.Sell)
	require.False(t, ok)

	got = got[:0]
	r8, err := e.Place(Order{TIF: GTC, Side: book.Sell, Price: 101, Qty: 3})
	require.NoError(t, err)
	r9, err := e.Place(Order{TIF: PostOnly, Side: book.Buy, Price: 101, Qty: 1})
	require.NoError(t, err)
	require.Equal(t, Rejected, r9.Status)
	require.Empty(t, got)

	r10, err := e.Place(Order{TIF: PostOnly, Side: book.Buy, Price: 100, Qty: 1})
	require.NoError(t, err)
	require.Equal(t, Open, r10.Status)

	got = got[:0]
	r11, err := e.Place(Order{TIF: GTC, Side: book.Buy, Price: 102, Qty: 5})
	require.NoError(t, err)
	require.Equal(t, PartiallyFilled, r11.Status)
	require.Equal(t, uint64(2), r11.RemainingQty)
	require.Equal(t, trades{{ID: 5, TakerOrderID: r11.OrderID, MakerOrderID: r8.OrderID, Price: 101, Qty: 3, TakerRemaining: 2, MakerRemaining: 0}}, got)
	price, totalQty, ok := e.book.Best(book.Buy)
	require.True(t, ok)
	require.Equal(t, uint64(102), price)
	require.Equal(t, uint64(2), totalQty)
}

func TestCancel(t *testing.T) {
	var got trades
	e, _ := New(10, 10, &got)

	r1, err := e.Place(Order{TIF: GTC, Side: book.Buy, Price: 100, Qty: 10})
	require.NoError(t, err)
	r2, err := e.Place(Order{TIF: GTC, Side: book.Sell, Price: 101, Qty: 7})
	require.NoError(t, err)
	r3, err := e.Place(Order{TIF: GTC, Side: book.Sell, Price: 100, Qty: 4})
	require.NoError(t, err)
	require.Equal(t, Filled, r3.Status)

	c, err := e.Cancel(r1.OrderID)
	require.NoError(t, err)
	require.Equal(t, Canceled, c.Status)
	require.Equal(t, r1.OrderID, c.OrderID)
	require.Equal(t, uint64(6), c.RemainingQty)
	_, _, ok := e.book.Best(book.Buy)
	require.False(t, ok)

	_, err = e.Cancel(r1.OrderID)
	require.ErrorIs(t, err, levels.ErrNotFound)

	_, err = e.Cancel(r3.OrderID)
	require.ErrorIs(t, err, levels.ErrNotFound)

	_, err = e.Cancel(0)
	require.ErrorIs(t, err, levels.ErrNotFound)

	c, err = e.Cancel(r2.OrderID)
	require.NoError(t, err)
	require.Equal(t, uint64(7), c.RemainingQty)
	_, _, ok = e.book.Best(book.Sell)
	require.False(t, ok)
}
