package cache_aside

import "caching-strategies/internal/repository/entity/order"

type CacheInterface[K uint64, V *order.Order] interface {
	Get(key K) (value *order.Order, ok bool)
	Add(key K, value *order.Order) (evicted bool)
}

type CacheAside struct {
	cache CacheInterface[uint64, *order.Order]
}

func New(cache CacheInterface[uint64, *order.Order]) *CacheAside {
	return &CacheAside{
		cache: cache,
	}
}

func (c *CacheAside) Get(key uint64) (value *order.Order, ok bool) {
	return c.cache.Get(key)
}

func (c *CacheAside) Add(key uint64, value *order.Order) (evicted bool) {
	return c.cache.Add(key, value)
}
