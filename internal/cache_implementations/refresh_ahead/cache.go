package refresh_ahead

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"time"
)

type CacheInterface[K uint64, V *order.Order] interface {
	Get(key K) (value *order.Order, ok bool)
	Add(key K, value *order.Order) (evicted bool)
}

type OrderRepoI interface {
	Save(ctx context.Context, order *order.Order) (uint64, error)
	Get(ctx context.Context, IDs []uint64) (map[uint64]order.Order, error)
}

type RefreshAheadCache struct {
	cache           CacheInterface[uint64, *order.Order]
	orderRepository OrderRepoI
	ttl             time.Duration
}

func New(cache CacheInterface[uint64, *order.Order], orderRepository OrderRepoI, ttl time.Duration) *RefreshAheadCache {
	return &RefreshAheadCache{
		cache:           cache,
		orderRepository: orderRepository,
		ttl:             ttl,
	}
}

func (c *RefreshAheadCache) Get(ctx context.Context, IDs []uint64) ([]order.Order, error) {
	refreshCacheCh := make(chan uint64, len(IDs))
	refreshCache := make([]uint64, 0, len(IDs))

	inCacheCh := make(chan order.Order, len(IDs))

	g := errgroup.Group{}
	g.SetLimit(100)

	// split requests to DB
	for _, ID := range IDs {
		ID := ID
		g.Go(func() error {
			value, ok := c.cache.Get(ID)
			if !ok || value == nil || value.ExpiredAt.After(time.Now().Add(-c.ttl/2)) {
				// нет в кэше или ttl скоро истекает - достаем из бд
				refreshCacheCh <- ID
				return nil
			}

			// получили значение из кэша
			inCacheCh <- *value

			return nil
		})
	}

	// never returns err
	_ = g.Wait()
	close(refreshCacheCh)
	close(inCacheCh)

	result := make([]order.Order, 0, len(IDs))
	// append cache to result
	for ord := range inCacheCh {
		result = append(result, ord)
	}

	// prepare for DB request
	for ID := range refreshCacheCh {
		refreshCache = append(refreshCache, ID)
	}

	fmt.Printf("#{len(result} items from aside cache")

	// обновляем данные в кэше
	if len(refreshCache) > 0 {
		ordersMap, err := c.orderRepository.Get(ctx, refreshCache)
		if err != nil {
			return nil, fmt.Errorf("err from repository: %s", err.Error())
		}

		for _, ord := range ordersMap {
			ord := ord
			ord.ExpiredAt = time.Now().Add(c.ttl)
			result = append(result, ord)

			g.Go(func() error {
				_ = c.cache.Add(ord.ID, &ord)

				return nil
			})
		}
		_ = g.Wait()

		fmt.Printf("#{len(ordersMap)} items from db")
	}

	return result, nil
}

func (c *RefreshAheadCache) Add(ctx context.Context, order *order.Order) error {
	orderID, err := c.orderRepository.Save(ctx, order)
	if err != nil {
		return errors.Wrap(err, "orderRepository.Save")
	}

	_ = c.cache.Add(orderID, order)

	return nil
}
