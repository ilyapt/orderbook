package matching

import (
	"math/rand"
	"testing"

	"orderbook/book"
)

// Run:
//
//	go test -run '^$' -bench . -benchmem -count 3 ./matching
//
// The book is prefilled with numLevels levels per side, ordersPerLevel
// orders of orderQty each, asks from mid+1 up and bids from mid-1 down.
// Every scenario keeps the book in that state: what a taker consumes is
// placed back inside the same iteration, so ns/op includes the
// replenishing Place calls where noted.

const (
	numLevels      = 100
	ordersPerLevel = 10
	orderQty       = 10
	mid            = 1000

	rndSize = 1 << 20
	rndMask = rndSize - 1
)

var rnd = func() []uint32 {
	r := rand.New(rand.NewSource(1))
	out := make([]uint32, rndSize)
	for i := range out {
		out[i] = r.Uint32()
	}
	return out
}()

var (
	sinkStatus Status
	sinkQty    uint64
)

type nopSink struct{}

func (nopSink) OnTrade(Trade) {}

func newEngine() *Engine {
	e, _ := New(4*numLevels*ordersPerLevel, 4*numLevels, nopSink{})
	for i := 1; i <= numLevels; i++ {
		for j := 0; j < ordersPerLevel; j++ {
			mustPlace(e, Order{TIF: GTC, Side: book.Sell, Price: mid + uint64(i), Qty: orderQty})
			mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid - uint64(i), Qty: orderQty})
		}
	}
	return e
}

func mustPlace(e *Engine, o Order) Report {
	r, err := e.Place(o)
	if err != nil {
		panic(err)
	}
	return r
}

// Scenario 1: a GTC order that rests and is cancelled. Onto the best bid
// level (push only) and onto a new price below the book (level created and
// dropped).
func BenchmarkRest(b *testing.B) {
	b.Run("existing-level", func(b *testing.B) {
		e := newEngine()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid - 1, Qty: orderQty})
			c, _ := e.Cancel(r.OrderID)
			sinkQty += c.RemainingQty
		}
	})
	b.Run("new-level", func(b *testing.B) {
		e := newEngine()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid - numLevels - 1, Qty: orderQty})
			c, _ := e.Cancel(r.OrderID)
			sinkQty += c.RemainingQty
		}
	})
}

// Scenario 2: a taker that fully fills the head of the best ask level, one
// trade, plus one replenishing Place.
func BenchmarkFillOne(b *testing.B) {
	e := newEngine()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid + 1, Qty: orderQty})
		sinkStatus = r.Status
		mustPlace(e, Order{TIF: GTC, Side: book.Sell, Price: mid + 1, Qty: orderQty})
	}
}

// Scenario 3: a taker that sweeps n orders across n/ordersPerLevel ask
// levels, plus n replenishing Place calls that recreate those levels.
func BenchmarkSweep(b *testing.B) {
	for _, n := range []int{10, 100} {
		lvls := n / ordersPerLevel
		b.Run("orders="+itoa(n), func(b *testing.B) {
			e := newEngine()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid + uint64(lvls), Qty: uint64(n) * orderQty})
				sinkStatus = r.Status
				for l := 1; l <= lvls; l++ {
					for j := 0; j < ordersPerLevel; j++ {
						mustPlace(e, Order{TIF: GTC, Side: book.Sell, Price: mid + uint64(l), Qty: orderQty})
					}
				}
			}
		})
	}
}

// Scenario 4: FOK for more than the whole ask side holds, rejected after
// walking every level without touching the book.
func BenchmarkFOKReject(b *testing.B) {
	e := newEngine()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := mustPlace(e, Order{TIF: FOK, Side: book.Buy, Price: mid + numLevels, Qty: numLevels*ordersPerLevel*orderQty + 1})
		sinkStatus = r.Status
	}
}

// Scenario 5: PostOnly that would cross, rejected on the first level.
func BenchmarkPostOnlyReject(b *testing.B) {
	e := newEngine()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := mustPlace(e, Order{TIF: PostOnly, Side: book.Buy, Price: mid + 1, Qty: orderQty})
		sinkStatus = r.Status
	}
}

// Scenario 6: a mix close to real flow: 70% rest and cancel, 25% single
// fill, 5% sweep of 10 orders, each with its replenishment.
func BenchmarkMixed(b *testing.B) {
	e := newEngine()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch op := rnd[i&rndMask] % 100; {
		case op < 70:
			r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid - 1, Qty: orderQty})
			c, _ := e.Cancel(r.OrderID)
			sinkQty += c.RemainingQty
		case op < 95:
			r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid + 1, Qty: orderQty})
			sinkStatus = r.Status
			mustPlace(e, Order{TIF: GTC, Side: book.Sell, Price: mid + 1, Qty: orderQty})
		default:
			r := mustPlace(e, Order{TIF: GTC, Side: book.Buy, Price: mid + 1, Qty: ordersPerLevel * orderQty})
			sinkStatus = r.Status
			for j := 0; j < ordersPerLevel; j++ {
				mustPlace(e, Order{TIF: GTC, Side: book.Sell, Price: mid + 1, Qty: orderQty})
			}
		}
	}
}

func itoa(n int) string {
	switch n {
	case 10:
		return "10"
	case 100:
		return "100"
	}
	return "?"
}
