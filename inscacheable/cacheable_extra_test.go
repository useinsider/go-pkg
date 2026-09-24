package inscacheable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetExistsDelete(t *testing.T) {
	c := Cacheable[string, string](nil, nil)

	t.Run("it_should_not_find_missing_key", func(t *testing.T) {
		assert.False(t, c.Exists("missing"))
	})

	t.Run("it_should_get_what_was_set", func(t *testing.T) {
		c.Set("k", "v", time.Minute)
		assert.True(t, c.Exists("k"))
		assert.Equal(t, "v", c.Get("k"))
	})

	t.Run("it_should_not_find_deleted_key", func(t *testing.T) {
		c.Set("gone", "v", time.Minute)
		c.Delete("gone")
		assert.False(t, c.Exists("gone"))
	})
}

func TestCache_Stop(t *testing.T) {
	t.Run("it_should_stop_cleanup_without_panicking", func(t *testing.T) {
		ttl := time.Minute
		c := Cacheable(func(k string) string { return k }, &ttl)
		assert.NotPanics(t, c.Stop)
	})
}

func TestCache_StopWithoutTTL(t *testing.T) {
	t.Run("it_should_block_forever_when_created_without_ttl", func(t *testing.T) {
		c := Cacheable[string, string](nil, nil)

		done := make(chan struct{})
		go func() {
			c.Stop()
			close(done)
		}()

		select {
		case <-done:
			t.Error("Stop() returned on a nil-ttl cache; the pinned deadlock has been fixed — update this test and the bug registry")
		case <-time.After(100 * time.Millisecond):
		}
	})
}
