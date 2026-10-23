package book

import (
	"math/rand"
	"slices"
	"strconv"
	"testing"

	"orderbook/levels"
)

// Run:
//
//	go test -run '^$' -bench . -benchmem -count 3 ./book
//
// Compares the blocked side against a flat sorted slice of the same
// entries with the same gallop search. Bid semantics: best is the
// highest price.

const (
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
	sinkInt  int
	sinkU64  uint64
	sinkBool bool
)

type flatSide struct {
	entries []entry
}

func (s *flatSide) find(price uint64) (int, levels.LevelIndex, bool) {
	i, ok := gallop(s.entries, true, price)
	if !ok {
		return i, -1, false
	}
	return i, levels.LevelIndex(s.entries[i].idx), true
}

func (s *flatSide) insertAt(i int, idx levels.LevelIndex, price uint64) {
	s.entries = slices.Insert(s.entries, i, entry{price: price, idx: int32(idx)})
}

func (s *flatSide) removeAt(i int) {
	s.entries = append(s.entries[:i], s.entries[i+1:]...)
}

// prices returns k distinct prices spread over a range four times larger,
// in random order, plus the leftover prices as a free pool.
func prices(k int) (live, free []uint64) {
	all := make([]uint64, 4*k)
	for i := range all {
		all[i] = uint64(i + 1)
	}
	rand.New(rand.NewSource(2)).Shuffle(len(all), func(a, b int) { all[a], all[b] = all[b], all[a] })
	return all[:k], all[k:]
}

func benchSide(live []uint64) *side {
	s := &side{bid: true}
	for i, p := range live {
		j, _, _ := s.find(p)
		s.insertAt(j, levels.LevelIndex(i), p)
	}
	return s
}

func benchFlat(live []uint64) *flatSide {
	s := &flatSide{}
	for i, p := range live {
		j, _, _ := s.find(p)
		s.insertAt(j, levels.LevelIndex(i), p)
	}
	return s
}

// spreadPrices returns k-10 static deep prices plus a 20-price region at the
// top of the book, half of it live and half free, so that churn stays within
// the ten best levels.
func spreadPrices(k int) (deep, live, free []uint64) {
	deep = make([]uint64, k-10)
	for i := range deep {
		deep[i] = uint64(i + 1)
	}
	region := make([]uint64, 20)
	for i := range region {
		region[i] = uint64(k - 10 + i + 1)
	}
	rand.New(rand.NewSource(3)).Shuffle(len(region), func(a, b int) { region[a], region[b] = region[b], region[a] })
	return deep, region[:10], region[10:]
}

var sizes = []int{100, 1_000, 10_000, 50_000}

func BenchmarkFind(b *testing.B) {
	for _, k := range sizes {
		live, _ := prices(k)
		b.Run("blocks/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchSide(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p, _, ok := s.find(live[rnd[i&rndMask]%uint32(k)])
				sinkInt += p.i
				sinkBool = ok
			}
		})
		b.Run("flat/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchFlat(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				j, _, ok := s.find(live[rnd[i&rndMask]%uint32(k)])
				sinkInt += j
				sinkBool = ok
			}
		})
	}
}

func BenchmarkFindNear(b *testing.B) {
	for _, k := range sizes {
		live, _ := prices(k)
		sorted := slices.Clone(live)
		slices.Sort(sorted)
		b.Run("blocks/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchSide(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p, _, ok := s.find(sorted[k-1-int(rnd[i&rndMask]%10)])
				sinkInt += p.i
				sinkBool = ok
			}
		})
		b.Run("flat/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchFlat(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				j, _, ok := s.find(sorted[k-1-int(rnd[i&rndMask]%10)])
				sinkInt += j
				sinkBool = ok
			}
		})
	}
}

func BenchmarkBest(b *testing.B) {
	for _, k := range sizes {
		live, _ := prices(k)
		b.Run("blocks/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchSide(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				l, ok := s.best()
				sinkInt += int(l)
				sinkBool = ok
			}
		})
		b.Run("flat/k="+strconv.Itoa(k), func(b *testing.B) {
			s := benchFlat(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sinkU64 += s.entries[len(s.entries)-1].price
			}
		})
	}
}

func BenchmarkChurn(b *testing.B) {
	for _, k := range sizes {
		b.Run("blocks/k="+strconv.Itoa(k), func(b *testing.B) {
			live, free := prices(k)
			s := benchSide(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a := int(rnd[i&rndMask] % uint32(k))
				f := int(rnd[(i+1)&rndMask] % uint32(len(free)))
				old, nw := live[a], free[f]
				p, _, _ := s.find(old)
				s.removeAt(p)
				p, _, _ = s.find(nw)
				s.insertAt(p, levels.LevelIndex(a), nw)
				live[a], free[f] = nw, old
			}
		})
		b.Run("flat/k="+strconv.Itoa(k), func(b *testing.B) {
			live, free := prices(k)
			s := benchFlat(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a := int(rnd[i&rndMask] % uint32(k))
				f := int(rnd[(i+1)&rndMask] % uint32(len(free)))
				old, nw := live[a], free[f]
				j, _, _ := s.find(old)
				s.removeAt(j)
				j, _, _ = s.find(nw)
				s.insertAt(j, levels.LevelIndex(a), nw)
				live[a], free[f] = nw, old
			}
		})
	}
}

func BenchmarkChurnSpread(b *testing.B) {
	for _, k := range sizes {
		b.Run("blocks/k="+strconv.Itoa(k), func(b *testing.B) {
			deep, live, free := spreadPrices(k)
			s := benchSide(append(deep, live...))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a := int(rnd[i&rndMask] % 10)
				f := int(rnd[(i+1)&rndMask] % 10)
				old, nw := live[a], free[f]
				p, _, _ := s.find(old)
				s.removeAt(p)
				p, _, _ = s.find(nw)
				s.insertAt(p, levels.LevelIndex(a), nw)
				live[a], free[f] = nw, old
			}
		})
		b.Run("flat/k="+strconv.Itoa(k), func(b *testing.B) {
			deep, live, free := spreadPrices(k)
			s := benchFlat(append(deep, live...))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				a := int(rnd[i&rndMask] % 10)
				f := int(rnd[(i+1)&rndMask] % 10)
				old, nw := live[a], free[f]
				j, _, _ := s.find(old)
				s.removeAt(j)
				j, _, _ = s.find(nw)
				s.insertAt(j, levels.LevelIndex(a), nw)
				live[a], free[f] = nw, old
			}
		})
	}
}

func BenchmarkDepth(b *testing.B) {
	live, _ := prices(1_000)
	for _, n := range []int{10, 50} {
		b.Run("cursor/n="+strconv.Itoa(n), func(b *testing.B) {
			s := benchSide(live)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var sum int
				cu, ok := s.cursor()
				for d := 0; d < n && ok; d++ {
					sum += int(cu.idx())
					ok = cu.prev()
				}
				sinkInt += sum
			}
		})
		b.Run("depth/n="+strconv.Itoa(n), func(b *testing.B) {
			s := benchSide(live)
			out := make([]levels.LevelIndex, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var sum int
				for _, l := range out[:s.depth(out)] {
					sum += int(l)
				}
				sinkInt += sum
			}
		})
		b.Run("flat/n="+strconv.Itoa(n), func(b *testing.B) {
			s := benchFlat(live)
			out := make([]levels.LevelIndex, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for d := range out {
					out[d] = levels.LevelIndex(s.entries[len(s.entries)-1-d].idx)
				}
				var sum int
				for _, l := range out {
					sum += int(l)
				}
				sinkInt += sum
			}
		})
	}
}
