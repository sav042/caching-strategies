package main

import (
	"caching-strategies/internal/cache_implementations/cache_aside"
	"caching-strategies/internal/cache_implementations/read_write_through_cache"
	repo "caching-strategies/internal/repository"
	"caching-strategies/internal/repository/entity/order"
	order_usecase "caching-strategies/internal/usecase/order"
	order_usecase_with_cache_aside "caching-strategies/internal/usecase/with_cache_aside"
	order_usecase_with_cache_through "caching-strategies/internal/usecase/with_cache_trough"
	"context"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"time"
)

func main() {
	ctx := context.Background()
	repository := repo.New()

	// make cache with 10ms TTL and 5 max keys
	cache := expirable.NewLRU[uint64, *order.Order](5, nil, time.Millisecond*10)
	asideCache := cache_aside.New(cache)
	readWriteThroughCache := read_write_through_cache.New(cache, repository)

	orderUsecase := order_usecase.New(repository)
	orderUsecaseWithCacheAside := order_usecase_with_cache_aside.New(repository, asideCache)
	orderUsecaseWithCacheThrough := order_usecase_with_cache_through.New(readWriteThroughCache)

	orderUsecase.Save(ctx, &order.Order{ID: 1})
	orderUsecase.Get(ctx, []uint64{1})

	orderUsecaseWithCacheAside.Save(ctx, &order.Order{ID: 2})
	orderUsecaseWithCacheAside.Get(ctx, []uint64{2})

	orderUsecaseWithCacheThrough.Add(ctx, &order.Order{ID: 3})
	orderUsecaseWithCacheThrough.Get(ctx, []uint64{3})
}
