package watcher

import (
	"caching-strategies/internal/repository/entity/order"
	"context"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"time"
)

const (
	watchTimeout = 10 * time.Millisecond
)

type CacheInterface[K uint64, V *order.Order] interface {
	Get(key K) (value *order.Order, ok bool)
	Add(key K, value *order.Order) (evicted bool)
}

type OrderRepoI interface {
	Save(ctx context.Context, order *order.Order) (uint64, error)
	Get(ctx context.Context, IDs []uint64) (map[uint64]order.Order, error)
}

type CacheRefresh struct {
	cache           CacheInterface[uint64, *order.Order]
	orderRepository OrderRepoI
	refreshCh       <-chan uint64
	cacheTTL        time.Duration
}

func New(
	cache CacheInterface[uint64, *order.Order],
	orderRepository OrderRepoI,
	refreshCh <-chan uint64,
	cacheTTL time.Duration,
) *CacheRefresh {
	return &CacheRefresh{
		cache:           cache,
		orderRepository: orderRepository,
		refreshCh:       refreshCh,
		cacheTTL:        cacheTTL,
	}
}

func (c *CacheRefresh) Start(ctx context.Context) {
	ticker := time.NewTicker(watchTimeout)
	defer ticker.Stop()

	// проверяем канал refreshCh c периодичностью watchTimeout
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(c.refreshCh) > 0 {
				chLen := len(c.refreshCh)
				IDs := make([]uint64, 0, chLen)

				// читаем из канала IDs которые нужно обновить в кэше
				for i := 0; i < chLen; i++ {
					IDs = append(IDs, <-c.refreshCh)
				}

				c.refresh(ctx, IDs)
			}
		}
	}
}

func (c *CacheRefresh) refresh(ctx context.Context, IDs []uint64) {
	start := time.Now()

	ordersMap, err := c.orderRepository.Get(ctx, IDs)
	if err != nil {
		log.Err(err).Msg("watcher.refresh error")
		return
	}

	g := errgroup.Group{}
	g.SetLimit(100)

	// обновляем хэш
	for _, ord := range ordersMap {
		ord := ord
		ord.ExpiredAt = time.Now().Add(c.cacheTTL)

		g.Go(func() error {
			_ = c.cache.Add(ord.ID, &ord)

			return nil
		})
	}
	_ = g.Wait()

	log.Info().
		Int("count", len(ordersMap)).
		Str("elapsed time", time.Since(start).String()).
		Msg("refresh orders in cache")
}
