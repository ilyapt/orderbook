package matching

import "orderbook/book"

type TIF uint8
type Status uint8

const (
	GTC TIF = iota + 1
	IOC
	FOK
	PostOnly
)
const (
	Open Status = iota + 1
	PartiallyFilled
	Filled
	Canceled
	Rejected
)

type Order struct {
	TIF   TIF
	Side  book.Side
	Price uint64
	Qty   uint64
}

type Report struct {
	OrderID      uint64
	RemainingQty uint64
	Status       Status
}

type Trade struct {
	ID             uint64
	TakerOrderID   uint64
	MakerOrderID   uint64
	Price          uint64
	Qty            uint64
	TakerRemaining uint64
	MakerRemaining uint64
}

type TradeSink interface {
	OnTrade(Trade)
}
