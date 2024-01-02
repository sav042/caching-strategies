package order_usecase_with_cache_through

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
)

type HotStorageI interface {
	Get(ctx context.Context, IDs []uint64) ([]order.Order, error)
	Add(ctx context.Context, order *order.Order) error
}

type Usecase struct {
	hotStorage HotStorageI
}

func New(hotStorage HotStorageI) *Usecase {
	return &Usecase{hotStorage: hotStorage}
}

func (uc *Usecase) Get(ctx context.Context, IDs []uint64) ([]order.Order, error) {
	return uc.hotStorage.Get(ctx, IDs)
}

func (uc *Usecase) Save(ctx context.Context, order *order.Order) error {
	return uc.hotStorage.Add(ctx, order)
}
