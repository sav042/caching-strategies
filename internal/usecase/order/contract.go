package order_usecase

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
)

type OrderRepoI interface {
	Save(ctx context.Context, order *order.Order) (uint64, error)
	Get(ctx context.Context, IDs []uint64) (map[uint64]order.Order, error)
}
