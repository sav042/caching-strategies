package main

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
	"github.com/hashicorp/golang-lru/v2/expirable"
	"time"
)

func main() {
	ctx := context.Background()
	repository := repo.New()
	ttl := time.Millisecond * 100

	// make cache with 100ms TTL and 5 max keys
	cache := expirable.NewLRU[uint64, *order.Order](5, nil, ttl)
	asideCache := cache_aside.New(cache)
	readWriteThroughCache := read_write_through_cache.New(cache, repository)
	refreshAheadCache := refresh_ahead.New(cache, repository, ttl)

	orderUsecase := order_usecase.New(repository)
	orderUsecaseWithCacheAside := order_usecase_with_cache_aside.New(repository, asideCache)
	orderUsecaseWithCacheThrough := order_usecase_with_cache_through.New(readWriteThroughCache)
	orderUsecaseWithCacheRefresh := order_usecase_with_cache_refresh.New(refreshAheadCache)

	orderUsecase.Save(ctx, &order.Order{ID: 1})
	orderUsecase.Get(ctx, []uint64{1})

	orderUsecaseWithCacheAside.Save(ctx, &order.Order{ID: 2})
	orderUsecaseWithCacheAside.Get(ctx, []uint64{2})

	orderUsecaseWithCacheThrough.Add(ctx, &order.Order{ID: 3})
	orderUsecaseWithCacheThrough.Get(ctx, []uint64{3})

	orderUsecaseWithCacheRefresh.Add(ctx, &order.Order{ID: 4})
	orderUsecaseWithCacheRefresh.Get(ctx, []uint64{4})
}
