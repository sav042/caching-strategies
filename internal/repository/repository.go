package repository

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
	"fmt"
	"sync"
	"time"
)

type Repo struct {
	DB sync.Map
}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) Get(ctx context.Context, IDs []uint64) (map[uint64]order.Order, error) {
	ordersMap := make(map[uint64]order.Order, len(IDs))

	for _, ID := range IDs {
		// mock db latency
		time.Sleep(1 * time.Millisecond)

		value, ok := r.DB.Load(ID)
		if !ok {
			return nil, fmt.Errorf("db loading error")
		}
		ord, ok := value.(order.Order)
		if !ok {
			return nil, fmt.Errorf("type casting error")
		}
		ordersMap[ord.ID] = ord
	}

	return ordersMap, nil
}

func (r *Repo) Save(ctx context.Context, order *order.Order) (uint64, error) {
	// mock db latency
	time.Sleep(1 * time.Millisecond)

	r.DB.Store(order.ID, *order)
	return order.ID, nil
}
