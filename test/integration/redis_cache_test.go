// Package integration_test wires two go-pkg modules together the way a
// consuming service does: insredis talking to a REAL Redis server, with an
// inscacheable cache in front of it.
//
// The dependency is a real `redis:7.4-alpine` container, not a stub — the
// workflow starts it and the test dials it at REDIS_ADDR (default
// localhost:6378, the same port the org's ucd-web integration job uses).
// There is deliberately no skip-when-absent branch: a missing Redis must fail
// the "Integration Tests" check, because a skipped test in a required check
// is a silent pass.
package integration_test

import (
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis"
	cacheable "github.com/useinsider/go-pkg/inscacheable"
	"github.com/useinsider/go-pkg/insredis"
)

// redisAddr is where the workflow publishes the container.
func redisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}

	return "localhost:6378"
}

// client initialises the package the way a service's bootstrap does, and
// fails the test — never skips — when the server is not reachable.
//
// insredis.Init keeps a package-level singleton, so every test in this binary
// shares one pool; keys are namespaced per test instead.
func client(t *testing.T) *redis.Client {
	t.Helper()

	c := insredis.Init(insredis.Config{
		RedisHost:   redisAddr(),
		DialTimeout: 5 * time.Second,
		ReadTimeout: 5 * time.Second,
	})

	if err := c.Ping().Err(); err != nil {
		t.Fatalf("PING %s: %v — the integration job must have a real Redis running", redisAddr(), err)
	}

	return c
}

// key namespaces a key to one test so a shared container cannot leak state
// between runs.
func key(t *testing.T, name string) string {
	t.Helper()

	return fmt.Sprintf("go-pkg:integration:%s:%s", t.Name(), name)
}

// TestInsRedisRoundTripsThroughRealServer exercises the behaviours only a real
// server has: server-side TTL bookkeeping, real key expiry, real type errors
// and a real pipeline round trip.
func TestInsRedisRoundTripsThroughRealServer(t *testing.T) {
	c := client(t)

	k := key(t, "value")
	t.Cleanup(func() { c.Del(k) })

	if err := c.Set(k, "hello", time.Minute).Err(); err != nil {
		t.Fatalf("SET: %v", err)
	}

	got, err := c.Get(k).Result()
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	if got != "hello" {
		t.Errorf("GET = %q, want %q", got, "hello")
	}

	// TTL is computed by the server. A stub would have to invent it.
	ttl, err := c.TTL(k).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}

	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("TTL = %v, want a positive value no greater than the 1m we set", ttl)
	}

	// PERSIST clears the expiry; -1 is Redis's "no TTL" sentinel.
	if err := c.Persist(k).Err(); err != nil {
		t.Fatalf("PERSIST: %v", err)
	}

	if ttl, err := c.TTL(k).Result(); err != nil || ttl != -time.Second {
		t.Errorf("TTL after PERSIST = %v (err %v), want -1s", ttl, err)
	}

	// A real server rejects INCR on a non-numeric value. This assertion is
	// what a hand-rolled in-memory fake gets wrong.
	if err := c.Incr(k).Err(); err == nil {
		t.Error("INCR on a non-numeric value returned no error")
	}

	// Real expiry: the server drops the key, and the miss comes back as
	// redis.Nil rather than an empty string.
	short := key(t, "short")
	t.Cleanup(func() { c.Del(short) })

	if err := c.Set(short, "gone soon", 100*time.Millisecond).Err(); err != nil {
		t.Fatalf("SET with short TTL: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if _, err := c.Get(short).Result(); err != redis.Nil { //nolint:errorlint // redis v6 returns the sentinel unwrapped
		t.Errorf("GET of an expired key = %v, want redis.Nil", err)
	}

	// One pipeline, one round trip, server-evaluated counter.
	counter := key(t, "counter")
	t.Cleanup(func() { c.Del(counter) })

	cmds, err := c.Pipelined(func(p redis.Pipeliner) error {
		p.Incr(counter)
		p.IncrBy(counter, 41)
		p.Expire(counter, time.Minute)

		return nil
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	if len(cmds) != 3 {
		t.Fatalf("pipeline returned %d replies, want 3", len(cmds))
	}

	if n, err := c.Get(counter).Result(); err != nil || n != "42" {
		t.Errorf("counter = %q (err %v), want \"42\"", n, err)
	}

	if n, err := c.Exists(counter, k).Result(); err != nil || n != 2 {
		t.Errorf("EXISTS = %d (err %v), want 2", n, err)
	}

	if err := c.Del(counter).Err(); err != nil {
		t.Fatalf("DEL: %v", err)
	}

	if n, err := c.Exists(counter).Result(); err != nil || n != 0 {
		t.Errorf("EXISTS after DEL = %d (err %v), want 0", n, err)
	}
}

// TestCacheableInFrontOfRealRedis is the shape a service actually ships:
// inscacheable as the hot path, insredis as the backing store. It asserts the
// cache spares Redis the repeat read, and that a miss after eviction reads
// whatever Redis holds *now* — which is only observable against a real store.
func TestCacheableInFrontOfRealRedis(t *testing.T) {
	c := client(t)

	var loads int64

	k := key(t, "backing")
	t.Cleanup(func() { c.Del(k) })

	if err := c.Set(k, "v1", time.Minute).Err(); err != nil {
		t.Fatalf("seed SET: %v", err)
	}

	ttl := time.Minute
	cache := cacheable.Cacheable(func(k string) string {
		atomic.AddInt64(&loads, 1)

		v, err := c.Get(k).Result()
		if err != nil {
			return "error: " + err.Error()
		}

		return v
	}, &ttl)

	defer cache.Stop()

	if got := cache.Get(k); got != "v1" {
		t.Fatalf("first read = %q, want v1", got)
	}

	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Fatalf("loader calls = %d, want 1", got)
	}

	// Change the value in Redis behind the cache's back. The cached read must
	// not see it; the read after eviction must.
	if err := c.Set(k, "v2", time.Minute).Err(); err != nil {
		t.Fatalf("second SET: %v", err)
	}

	if got := cache.Get(k); got != "v1" {
		t.Errorf("cached read = %q, want the stale v1 — the cache went to Redis", got)
	}

	if got := atomic.LoadInt64(&loads); got != 1 {
		t.Errorf("loader calls after cached read = %d, want it unchanged at 1", got)
	}

	cache.Delete(k)

	if got := cache.Get(k); got != "v2" {
		t.Errorf("read after eviction = %q, want v2 from Redis", got)
	}

	if got := atomic.LoadInt64(&loads); got != 2 {
		t.Errorf("loader calls after eviction = %d, want 2", got)
	}

	// A key Redis does not have surfaces the miss rather than a plausible
	// blank. Cacheable's loader-backed Get is why the sentinel is visible at
	// all here.
	missing := key(t, "absent-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if got := cache.Get(missing); got != "error: "+redis.Nil.Error() {
		t.Errorf("read of an absent key = %q, want the redis.Nil miss", got)
	}
}
