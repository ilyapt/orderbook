package levels_test

import (
	"container/list"
	"math/rand"
	"runtime"
	"strconv"
	"testing"

	"orderbook/levels"
)

// Run:
//
//	go test -run '^$' -bench . -benchmem -count 3 ./levels
//
// b.N is always a number of operations, so ns/op is directly ns/push,
// ns/cancel or ns/(cancel+push). Scenario 1 (Push) and 2 (Cancel) run in
// batches of `batch` operations on a fresh structure each, rebuilt outside
// the timer: the arena index is 24-bit and b.N can reach hundreds of
// millions, so one structure for all of b.N would overflow it and eat
// gigabytes. Scenario 3 (Steady) keeps a fixed number of live orders; each
// op is one cancel plus one push, and GC activity during the timed section
// is reported per operation.

// Reference implementation of the levels layer built the idiomatic way:
// container/list per level plus a map from order id to the list element
// for O(1) cancel. This is the structure used by
// github.com/i25959341/orderbook, minus its numeric types and string keys.
type refOrder struct {
	id    uint64
	qty   uint64
	level int
}

type refLevel struct {
	price    uint64
	totalQty uint64
	orders   *list.List
}

type ref struct {
	levels []*refLevel
	byID   map[uint64]*list.Element
	nextID uint64
}

func makeRef(ordersCap, levelsCap int) *ref {
	return &ref{
		levels: make([]*refLevel, 0, levelsCap),
		byID:   make(map[uint64]*list.Element, ordersCap),
	}
}

func (r *ref) NewLevel(price uint64) int {
	r.levels = append(r.levels, &refLevel{price: price, orders: list.New()})
	return len(r.levels) - 1
}

func (r *ref) Push(level int, qty uint64) uint64 {
	r.nextID++
	id := r.nextID
	lvl := r.levels[level]
	e := lvl.orders.PushBack(&refOrder{id: id, qty: qty, level: level})
	lvl.totalQty += qty
	r.byID[id] = e
	return id
}

func (r *ref) Cancel(id uint64) (level int, empty bool, ok bool) {
	e, ok := r.byID[id]
	if !ok {
		return 0, false, false
	}
	delete(r.byID, id)
	o := e.Value.(*refOrder)
	lvl := r.levels[o.level]
	lvl.orders.Remove(e)
	lvl.totalQty -= o.qty
	return o.level, lvl.orders.Len() == 0, true
}

const (
	numLevels = 1000
	batch     = 100_000

	rndSize = 1 << 20
	rndMask = rndSize - 1
)

// Pre-generated randomness so that RNG cost is identical and negligible
// for both implementations.
var rnd = func() []uint32 {
	r := rand.New(rand.NewSource(1))
	out := make([]uint32, rndSize)
	for i := range out {
		out[i] = r.Uint32()
	}
	return out
}()

var (
	sinkHandle levels.Handle
	sinkID     uint64
	sinkLevel  int
)

func newLevels(ordersCap int) (*levels.Levels, []levels.LevelIndex) {
	l := levels.New(ordersCap, numLevels)
	lv := make([]levels.LevelIndex, numLevels)
	for i := range lv {
		lv[i] = l.NewLevel(uint64(i + 1))
	}
	return l, lv
}

func newRef(ordersCap int) (*ref, []int) {
	r := makeRef(ordersCap, numLevels)
	lv := make([]int, numLevels)
	for i := range lv {
		lv[i] = r.NewLevel(uint64(i + 1))
	}
	return r, lv
}

func shuffle[T any](s []T) {
	rand.New(rand.NewSource(2)).Shuffle(len(s), func(a, c int) { s[a], s[c] = s[c], s[a] })
}

// Scenario 1: push orders onto random pre-existing levels, `batch` at a
// time onto a fresh structure.
func BenchmarkPush(b *testing.B) {
	b.Run("levels", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i += batch {
			n := min(batch, b.N-i)
			b.StopTimer()
			l, lv := newLevels(n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				h, _ := l.Push(lv[rnd[j&rndMask]%numLevels], 1)
				sinkHandle += h
			}
		}
	})
	b.Run("ref", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i += batch {
			n := min(batch, b.N-i)
			b.StopTimer()
			r, lv := newRef(n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				sinkID += r.Push(lv[rnd[j&rndMask]%numLevels], 1)
			}
		}
	})
}

// Scenario 2: cancel resting orders in random order, `batch` at a time on
// a fresh structure.
func BenchmarkCancel(b *testing.B) {
	b.Run("levels", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i += batch {
			n := min(batch, b.N-i)
			b.StopTimer()
			l, lv := newLevels(n)
			hs := make([]levels.Handle, n)
			for j := range hs {
				hs[j], _ = l.Push(lv[rnd[j&rndMask]%numLevels], 1)
			}
			shuffle(hs)
			b.StartTimer()
			for _, h := range hs {
				lvl, _, _ := l.Cancel(h)
				sinkLevel += int(lvl)
			}
		}
	})
	b.Run("ref", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i += batch {
			n := min(batch, b.N-i)
			b.StopTimer()
			r, lv := newRef(n)
			ids := make([]uint64, n)
			for j := range ids {
				ids[j] = r.Push(lv[rnd[j&rndMask]%numLevels], 1)
			}
			shuffle(ids)
			b.StartTimer()
			for _, id := range ids {
				lvl, _, _ := r.Cancel(id)
				sinkLevel += lvl
			}
		}
	})
}

// Scenario 3: steady state. Keep `live` orders resting; each op cancels a
// random one and pushes a replacement onto a random level. GC count and
// pause time during the timed section are reported per operation.
func BenchmarkSteady(b *testing.B) {
	for _, live := range []int{10_000, 100_000, 1_000_000} {
		live := live
		name := "live=" + strconv.Itoa(live)
		b.Run("levels/"+name, func(b *testing.B) {
			l, lv := newLevels(live)
			hs := make([]levels.Handle, live)
			for j := range hs {
				hs[j], _ = l.Push(lv[rnd[j&rndMask]%numLevels], 1)
			}
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				k := int(rnd[i&rndMask]) % live
				lvl, _, _ := l.Cancel(hs[k])
				sinkLevel += int(lvl)
				hs[k], _ = l.Push(lv[rnd[(i+1)&rndMask]%numLevels], 1)
			}
			b.StopTimer()
			runtime.ReadMemStats(&after)
			reportGC(b, &before, &after)
		})
		b.Run("ref/"+name, func(b *testing.B) {
			r, lv := newRef(live)
			ids := make([]uint64, live)
			for j := range ids {
				ids[j] = r.Push(lv[rnd[j&rndMask]%numLevels], 1)
			}
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				k := int(rnd[i&rndMask]) % live
				lvl, _, _ := r.Cancel(ids[k])
				sinkLevel += lvl
				ids[k] = r.Push(lv[rnd[(i+1)&rndMask]%numLevels], 1)
			}
			b.StopTimer()
			runtime.ReadMemStats(&after)
			reportGC(b, &before, &after)
		})
	}
}

// reportGC normalises GC activity per benchmark operation so that runs with
// different b.N are comparable.
func reportGC(b *testing.B, before, after *runtime.MemStats) {
	n := float64(b.N)
	b.ReportMetric(float64(after.NumGC-before.NumGC)/n*1e6, "gcs/Mop")
	b.ReportMetric(float64(after.PauseTotalNs-before.PauseTotalNs)/n, "gc-pause-ns/op")
}
