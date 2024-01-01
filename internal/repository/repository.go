package repository

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
	"strconv"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (r Repo) Get(ctx context.Context, IDs []uint64) (map[uint64]order.Order, error) {
	// get data from db
	ordersMap := make(map[uint64]order.Order, len(IDs))
	// fill with mock data
	for i := 0; i < len(IDs); i++ {
		ordersMap[uint64(i)] = order.Order{
			ID:   uint64(i),
			Item: strconv.Itoa(i),
		}
	}
	return ordersMap, nil
}

func (r Repo) Save(ctx context.Context, order *order.Order) (uint64, error) {
	return order.ID, nil
}
