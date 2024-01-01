package order_usecase_with_cache_aside

import (
	"caching-strategies/internal/repository"
	"caching-strategies/internal/repository/entity/order"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"caching-strategies/internal/cache_implementations/cache_aside"
)

type Usecase struct {
	repo  *repository.Repo
	cache *cache_aside.CacheAside
}

func New(repo *repository.Repo, cache *cache_aside.CacheAside) *Usecase {
	return &Usecase{
		repo:  repo,
		cache: cache,
	}
}

func (uc *Usecase) Get(ctx context.Context, IDs []uint64) ([]order.Order, error) {
	notInCacheCh := make(chan uint64, len(IDs))
	notInCache := make([]uint64, 0, len(IDs))

	inCacheCh := make(chan order.Order, len(IDs))

	g := errgroup.Group{}
	g.SetLimit(100)

	// split requests to DB
	for _, ID := range IDs {
		ID := ID
		g.Go(func() error {
			value, ok := uc.cache.Get(ID)
			if !ok || value == nil {
				// нет в кэше, будем искать в бд
				notInCacheCh <- ID

				return nil
			}

			// получили значение из кэша
			inCacheCh <- *value

			return nil
		})
	}

	// never returns err
	_ = g.Wait()
	close(notInCacheCh)
	close(inCacheCh)

	result := make([]order.Order, 0, len(IDs))
	// append cache to result
	for ord := range inCacheCh {
		result = append(result, ord)
	}

	// prepare for DB request
	for ID := range notInCacheCh {
		notInCache = append(notInCache, ID)
	}

	fmt.Printf("#{len(result} items from aside cache")

	// обновляем данные в кэше
	if len(notInCache) > 0 {
		ordersMap, err := uc.repo.Get(ctx, notInCache)
		if err != nil {
			return nil, fmt.Errorf("err from repository: %s", err.Error())
		}

		for _, ord := range ordersMap {
			ord := ord
			result = append(result, ord)

			g.Go(func() error {
				_ = uc.cache.Add(ord.ID, &ord)

				return nil
			})
		}
		_ = g.Wait()

		fmt.Printf("#{len(ordersMap)} items from db")
	}

	return result, nil
}

func (uc *Usecase) Save(ctx context.Context, order *order.Order) error {
	orderID, err := uc.repo.Save(ctx, order)
	if err != nil {
		return errors.Wrap(err, "repo.Save")
	}

	order.ID = orderID

	_ = uc.cache.Add(orderID, order)

	fmt.Printf("cache aside updated: #{*order}\n")

	return nil
}
