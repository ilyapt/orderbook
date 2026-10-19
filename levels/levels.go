package levels

import (
	"errors"
)

const (
	NoHandle Handle = 0

	indexBits = 24
	indexMask = 1<<indexBits - 1
	seqMax    = 1<<(64-indexBits) - 1
)

var (
	ErrNotFound = errors.New("levels: order not found")
	ErrZeroQty  = errors.New("levels: qty must be positive")
)

type Handle uint64

func makeHandle(index int32, seq uint64) Handle {
	return Handle(seq<<indexBits | uint64(index))
}

func SeqHandle(seq uint64) Handle {
	return makeHandle(indexMask, seq)
}

func (h Handle) index() int32 { return int32(uint64(h) & indexMask) }
func (h Handle) Seq() uint64  { return uint64(h) >> indexBits }

type LevelIndex int32

type level struct {
	price    uint64
	totalQty uint64

	head int32
	tail int32
}

type Levels struct {
	orders    []order
	freeOrder int32

	levels    []level
	freeLevel int32
}

func New(ordersCap, levelsCap int) *Levels {
	return &Levels{
		orders:    make([]order, 0, ordersCap),
		freeOrder: -1,
		levels:    make([]level, 0, levelsCap),
		freeLevel: -1,
	}
}

func (l *Levels) alloc() int32 {
	if l.freeOrder >= 0 {
		index := l.freeOrder
		l.freeOrder = l.orders[l.freeOrder].next
		l.orders[index].next = -1
		return index
	}
	if len(l.orders) == indexMask {
		panic("levels: alloc: order capacity exhausted")
	}
	index := int32(len(l.orders))
	l.orders = append(l.orders, order{qty: 0, prev: -1, next: -1})
	return index
}

func (l *Levels) free(index int32) {
	o := &l.orders[index]
	o.level = -1
	o.qty = 0
	o.prev = -1
	o.next = l.freeOrder
	l.freeOrder = index
}

func (l *Levels) NewLevel(price uint64) LevelIndex {
	if l.freeLevel >= 0 {
		index := l.freeLevel
		l.freeLevel = l.levels[index].head
		l.levels[index].head = -1
		l.levels[index].tail = -1
		l.levels[index].price = price
		return LevelIndex(index)
	}
	index := int32(len(l.levels))
	l.levels = append(l.levels, level{price: price, totalQty: 0, head: -1, tail: -1})
	return LevelIndex(index)
}

func (l *Levels) FreeLevel(level LevelIndex) {
	lvl := &l.levels[level]
	if lvl.tail == -2 {
		panic("levels: FreeLevel: level already freed")
	}
	if lvl.head != -1 || lvl.tail != -1 || lvl.totalQty != 0 {
		panic("levels: FreeLevel: level is not empty")
	}
	lvl.price = 0
	lvl.totalQty = 0
	lvl.tail = -2
	lvl.head = l.freeLevel
	l.freeLevel = int32(level)
}
