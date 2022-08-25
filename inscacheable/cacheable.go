package inscacheable

import (
	"github.com/jellydator/ttlcache/v3"
	"time"
)

type Cacher[K comparable, V any] interface {
	Get(k K) V
	Set(k K, v V, ttl time.Duration)
}

type Cache[K comparable, V any] struct {
	cache *ttlcache.Cache[K, V]
}

func (c *Cache[K, V]) Get(k K) V {
	return c.cache.Get(k).Value()
}

func (c *Cache[K, V]) Set(k K, v V, ttl time.Duration) {
	c.cache.Set(k, v, ttl)
}

// Cacheable is the main function that should be used as
// func getter(key string) string { ... the original getter function ... }
// var ttl = 1 * time.Minute
// var cache = cacheable.Cacheable(getter, &ttl)
// func Get(key string) string { return cache.get(key) }
func Cacheable[K comparable, V any](getter func(key K) V, ttl *time.Duration) Cache[K, V] {
	return Cache[K, V]{makeCache(ttl, makeLoader(getter))}
}

func makeCache[K comparable, V any](ttl *time.Duration, loader ttlcache.LoaderFunc[K, V]) *ttlcache.Cache[K, V] {
	var options []ttlcache.Option[K, V]

	if ttl != nil {
		options = append(options, ttlcache.WithTTL[K, V](*ttl))
	}

	if loader != nil {
		options = append(options, ttlcache.WithLoader[K, V](loader))
	}

	var cache = ttlcache.New[K, V](
		options...,
	)

	if ttl != nil {
		go cache.Start()
	}

	return cache
}

func makeLoader[K comparable, V any](getter func(key K) V) ttlcache.LoaderFunc[K, V] {
	fn :=
		func(c *ttlcache.Cache[K, V], key K) *ttlcache.Item[K, V] {
			var v = getter(key)

			item := c.Set(key, v, ttlcache.DefaultTTL)
			return item
		}
	return fn
}
