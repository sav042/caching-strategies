# Caching strategies example

https://hazelcast.com/blog/a-hitchhikers-guide-to-caching-patterns/

https://www.prisma.io/dataguide/managing-databases/introduction-database-caching

https://www.youtube.com/watch?v=AFGC8ci5jDk

## Cache aside / Read Write Through

Read from cache -> if cache miss, read from db

Write to db -> write to cache

When to use: many R/W

Pros: only active data in cache

Cons: many cache misses reduce performance

## Refresh ahead

If data is near expiration -> async refresh cache

When to use: many reads, few writes

Pros: low latency, better performance for peaks

Cons: inconsistency risks (can be fixed with write-through/write-behind strategies)

## Write around

Write to DB -> async update cache

When to use: data is only written once and not updated

Pros: DB is always consistent, requires fewer cache resources

Cons: stale cache, write latency

## Write behind

Write to cache -> async update DB

When to use: many reads, few writes

Pros: few DB load, low latency

Cons: can lose updates, eventual consistency (not strong)
