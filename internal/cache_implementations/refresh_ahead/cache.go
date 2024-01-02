package refresh_ahead

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"time"
)

const refreshFactor = 2

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
	TTL             time.Duration
	refreshCh       chan<- uint64
}

func New(
	cache CacheInterface[uint64, *order.Order],
	orderRepository OrderRepoI,
	ttl time.Duration,
	refreshCh chan<- uint64,
) *RefreshAheadCache {
	return &RefreshAheadCache{
		cache:           cache,
		orderRepository: orderRepository,
		TTL:             ttl,
		refreshCh:       refreshCh,
	}
}

func (c *RefreshAheadCache) Get(ctx context.Context, IDs []uint64) ([]order.Order, error) {
	notInCacheCh := make(chan uint64, len(IDs))
	notInCache := make([]uint64, 0, len(IDs))

	inCacheCh := make(chan order.Order, len(IDs))

	g := errgroup.Group{}
	g.SetLimit(100)

	// split requests to DB
	for _, ID := range IDs {
		ID := ID
		g.Go(func() error {
			value, ok := c.cache.Get(ID)
			if !ok || value == nil {
				// нет в кэше, будем искать в бд
				notInCacheCh <- ID

				return nil
			}

			// если ttl кэша уменьшился в refreshFactor раз - пишем в канал обновления
			if value.ExpiredAt.Sub(time.Now()) <= c.TTL/refreshFactor {
				// если канал полный - не пишем, чтобы не заблокироваться
				if len(c.refreshCh) < cap(c.refreshCh) {
					c.refreshCh <- ID
				} else {
					log.Warn().Msg("refreshCh is full")
				}
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

	log.Debug().Msgf("get items from cache", "count", len(IDs))

	// обновляем данные в кэше
	if len(notInCache) > 0 {
		ordersMap, err := c.orderRepository.Get(ctx, notInCache)
		if err != nil {
			return nil, fmt.Errorf("err from repository: %s", err.Error())
		}

		for _, ord := range ordersMap {
			ord := ord
			result = append(result, ord)

			g.Go(func() error {
				_ = c.cache.Add(ord.ID, &ord)

				return nil
			})
		}
		_ = g.Wait()

		log.Debug().Msgf("get from db", "count", len(ordersMap))
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
