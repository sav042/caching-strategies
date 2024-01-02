package usecases

import (
	"caching-strategies/internal/cache_implementations/cache_aside"
	"caching-strategies/internal/cache_implementations/read_write_through"
	"caching-strategies/internal/cache_implementations/refresh_ahead"
	repo "caching-strategies/internal/repository"
	"caching-strategies/internal/repository/entity/order"
	order_usecase "caching-strategies/internal/usecases/0_without_cache"
	order_usecase_with_cache_aside "caching-strategies/internal/usecases/1_cache_aside"
	order_usecase_with_cache_through "caching-strategies/internal/usecases/2_read_write_through"
	order_usecase_with_cache_refresh "caching-strategies/internal/usecases/3_refresh_ahead"
	"caching-strategies/internal/watcher"
	"context"
	"fmt"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

const (
	cacheSize    = 1000
	cacheTTL     = 3 * time.Second
	ordersNumber = 1000
	batchSize    = 1
	ctxTimeout   = 10 * time.Second
)

type UsecaseI interface {
	Get(ctx context.Context, IDs []uint64) ([]order.Order, error)
	Save(ctx context.Context, order *order.Order) error
}

func setup(ctx context.Context) (*repo.Repo, *expirable.LRU[uint64, *order.Order]) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	repository := repo.New()
	cache := expirable.NewLRU[uint64, *order.Order](cacheSize, nil, cacheTTL)
	for i := 0; i < ordersNumber; i++ {
		_, err := repository.Save(ctx, &order.Order{ID: uint64(i)})
		if err != nil {
			panic(err.Error())
		}
	}
	return repository, cache
}

func getOrders(ctx context.Context, N uint64, uc UsecaseI) {
	start := time.Now()

	var i uint64
	for ; i < N; i++ {
		_, err := uc.Get(ctx, []uint64{i})
		if err != nil {
			panic(err.Error())
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("getOrders timeout: %s\n", elapsed)
}

// without cache ~ 1.2 sec
func TestWithoutCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	repository, _ := setup(ctx)
	usecase := order_usecase.New(repository)

	// cold cache
	getOrders(ctx, ordersNumber, usecase)
	// cold cache
	getOrders(ctx, ordersNumber, usecase)
}

// with warm cache ~ 6 msec
func TestCacheAside(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	repository, cache := setup(ctx)
	asideCache := cache_aside.New(cache)
	usecase := order_usecase_with_cache_aside.New(repository, asideCache)

	// cold cache
	getOrders(ctx, ordersNumber, usecase)
	// warm cache
	getOrders(ctx, ordersNumber, usecase)
}

// with warm cache ~ 6 msec
func TestCacheThrough(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	repository, cache := setup(ctx)
	readWriteThroughCache := read_write_through.New(cache, repository)
	usecase := order_usecase_with_cache_through.New(readWriteThroughCache)

	// cold cache
	getOrders(ctx, ordersNumber, usecase)
	// warm cache
	getOrders(ctx, ordersNumber, usecase)
}

// with warm cache ~ 6 msec
// after expiring ~ 1.2 sec
func TestCacheThroughWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	repository, cache := setup(ctx)
	readWriteThroughCache := read_write_through.New(cache, repository)
	usecase := order_usecase_with_cache_through.New(readWriteThroughCache)

	// cold cache
	getOrders(ctx, ordersNumber, usecase)
	time.Sleep(cacheTTL / 2)

	// warm cache
	getOrders(ctx, ordersNumber, usecase)
	time.Sleep(cacheTTL / 2)

	// cache was expired
	getOrders(ctx, ordersNumber, usecase)
}

// warm cache ~ 5 msec
// refreshed cache ~ 9 msec
func TestCacheRefreshAhead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	repository, cache := setup(ctx)

	refreshCh := make(chan uint64, 1000)
	defer close(refreshCh)

	// start cache-refresh watcher
	cacheWatcher := watcher.New(cache, repository, refreshCh, cacheTTL)
	go cacheWatcher.Start(ctx)

	refreshAheadCache := refresh_ahead.New(cache, repository, cacheTTL, refreshCh)
	usecase := order_usecase_with_cache_refresh.New(refreshAheadCache)

	// cold cache
	getOrders(ctx, ordersNumber, usecase)
	time.Sleep(cacheTTL / 2)

	// warming cache while reading
	getOrders(ctx, ordersNumber, usecase)
	time.Sleep(cacheTTL / 2)

	// cache wasn't expired
	getOrders(ctx, ordersNumber, usecase)
}
