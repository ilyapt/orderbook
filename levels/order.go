package levels

type order struct {
	qty   uint64
	seq   uint64
	level LevelIndex
	prev  int32
	next  int32
}

func (l *Levels) Push(level LevelIndex, seq, qty uint64) (Handle, error) {
	if qty == 0 {
		return NoHandle, ErrZeroQty
	}

	lvl := l.getLevel(level)

	if seq == 0 || seq > seqMax {
		panic("levels: Push: invalid seq")
	}

	index := l.alloc()
	o := &l.orders[index]

	o.level = level
	o.qty = qty
	o.seq = seq

	lvl.totalQty += qty

	if lvl.head == -1 {
		lvl.head = index
		lvl.tail = index

		return makeHandle(index, seq), nil
	}

	o.prev = lvl.tail
	l.orders[lvl.tail].next = index
	lvl.tail = index
	return makeHandle(index, seq), nil
}

func (l *Levels) Cancel(h Handle) (LevelIndex, uint64, bool, error) {
	index, o, err := l.get(h)
	if err != nil {
		return LevelIndex(0), 0, false, err
	}
	qty := o.qty
	levelId, empty := l.remove(index, o)
	return levelId, qty, empty, nil
}

func (l *Levels) Fill(h Handle, qty uint64) (LevelIndex, uint64, bool) {
	index, o, err := l.get(h)
	if err != nil {
		panic("levels: Fill: stale handle")
	}
	if o.qty < qty {
		panic("levels: Fill: qty exceeds order remainder")
	}
	if o.qty > qty {
		o.qty -= qty
		l.levels[o.level].totalQty -= qty
		return o.level, o.qty, false
	}

	idx, empty := l.remove(index, o)
	return idx, 0, empty
}

func (l *Levels) remove(index int32, o *order) (LevelIndex, bool) {

	levelId := o.level
	lvl := &l.levels[o.level]

	if o.qty > lvl.totalQty {
		panic("levels: remove: order qty exceeds level totalQty")
	}
	lvl.totalQty -= o.qty

	if lvl.head == index {
		lvl.head = o.next
	}
	if lvl.tail == index {
		lvl.tail = o.prev
	}
	if o.prev != -1 {
		l.orders[o.prev].next = o.next
	}
	if o.next != -1 {
		l.orders[o.next].prev = o.prev
	}
	l.free(index)

	empty := lvl.head == -1
	if (lvl.tail == -1) != empty || (lvl.totalQty == 0) != empty {
		panic("levels: remove: level invariant broken (head/tail/totalQty mismatch)")
	}
	return levelId, lvl.totalQty == 0
}

func (l *Levels) get(h Handle) (int32, *order, error) {
	index := h.index()
	if int(index) >= len(l.orders) {
		return 0, nil, ErrNotFound
	}
	o := &l.orders[index]
	if o.level < 0 || o.seq != h.Seq() {
		return 0, nil, ErrNotFound
	}
	return index, o, nil
}
