package levels

func (l *Levels) Front(lvl LevelIndex) (Handle, bool) {
	index := l.getLevel(lvl).head
	if index < 0 {
		return NoHandle, false
	}

	o := &l.orders[index]
	return makeHandle(index, o.seq), true
}

func (l *Levels) getLevel(lvl LevelIndex) *level {
	if lvl < 0 || lvl >= LevelIndex(len(l.levels)) {
		panic("levels: level index out of range")
	}
	if l.levels[lvl].tail == -2 {
		panic("levels: level is freed")
	}
	return &l.levels[lvl]
}

func (l *Levels) Price(lvl LevelIndex) uint64 {
	return l.getLevel(lvl).price
}

func (l *Levels) TotalQty(lvl LevelIndex) uint64 {
	return l.getLevel(lvl).totalQty
}

func (l *Levels) Qty(h Handle) (uint64, error) {
	_, o, err := l.get(h)
	if err != nil {
		return 0, err
	}
	return o.qty, nil
}

func (l *Levels) Level(h Handle) (LevelIndex, error) {
	_, o, err := l.get(h)
	if err != nil {
		return 0, err
	}
	return o.level, nil
}
