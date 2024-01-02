package order

import "time"

type Order struct {
	ID        uint64
	Item      string
	ExpiredAt time.Time
}
