package insredis

import (
	"testing"
	"time"
)

// Init keeps a package-level singleton; subtests run in order against it.
// No test talks to a real Redis — go-redis connects lazily.

func TestInit(t *testing.T) {
	redisClient = nil

	t.Run("it_should_apply_defaults_for_zero_config", func(t *testing.T) {
		client := Init(Config{RedisHost: "localhost:6379"})
		if client == nil {
			t.Fatal("Init() = nil, want client")
		}

		opts := client.Options()
		if opts.Addr != "localhost:6379" {
			t.Errorf("Addr = %q, want localhost:6379", opts.Addr)
		}
		if opts.PoolSize != 10 {
			t.Errorf("PoolSize = %d, want default 10", opts.PoolSize)
		}
		if opts.DialTimeout != 500*time.Millisecond {
			t.Errorf("DialTimeout = %v, want default 500ms", opts.DialTimeout)
		}
		if opts.ReadTimeout != 500*time.Millisecond {
			t.Errorf("ReadTimeout = %v, want default 500ms", opts.ReadTimeout)
		}
		if opts.MaxRetries != 0 {
			t.Errorf("MaxRetries = %d, want 0 (no default applied)", opts.MaxRetries)
		}
	})

	t.Run("it_should_return_cached_client_and_ignore_new_config", func(t *testing.T) {
		first := GetClient()

		client := Init(Config{RedisHost: "other-host:9999", RedisPoolSize: 99})
		if client != first {
			t.Error("Init() second call returned a new client, want the cached singleton")
		}
		if client.Options().Addr != "localhost:6379" {
			t.Errorf("Addr = %q, want the original localhost:6379 (new config ignored)", client.Options().Addr)
		}
	})
}

func TestGetClient(t *testing.T) {
	t.Run("it_should_return_the_singleton", func(t *testing.T) {
		if GetClient() != redisClient {
			t.Error("GetClient() != redisClient")
		}
	})
}
