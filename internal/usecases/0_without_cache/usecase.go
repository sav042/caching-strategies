package order_usecase

import (
	"caching-strategies/internal/repository"
	"caching-strategies/internal/repository/entity/order"
	"context"
	"fmt"
)

type Usecase struct {
	repo *repository.Repo
}

func New(repo *repository.Repo) *Usecase {
	return &Usecase{repo: repo}
}

func (uc *Usecase) Get(ctx context.Context, IDs []uint64) ([]order.Order, error) {
	ordersMap, err := uc.repo.Get(ctx, IDs)
	if err != nil {
		return nil, fmt.Errorf("err from repository: %s", err.Error())
	}

	result := make([]order.Order, 0, len(IDs))
	for _, ord := range ordersMap {
		result = append(result, ord)
	}

	return result, nil
}

func (uc *Usecase) Save(ctx context.Context, order *order.Order) error {
	if _, err := uc.repo.Save(ctx, order); err != nil {
		return err
	}
	return nil
}
