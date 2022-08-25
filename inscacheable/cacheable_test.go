package inscacheable

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type CustomResponse struct {
	value *string
	err   error
}

func TestCachableLoader(t *testing.T) {
	counter := 0
	getter := func(key string) CustomResponse {
		if counter == 3 {
			return CustomResponse{nil, errors.New("items depleted")}
		}
		counter++
		value := counter
		msg := fmt.Sprintf("key %s is %d", key, value)
		return CustomResponse{&msg, nil}
	}

	ttl := 1 * time.Second
	c := Cacheable(getter, &ttl)

	t.Run("it_should_get_the_first_item", func(t *testing.T) {
		actual1 := *c.Get("A").value
		assert.Equal(t, "key A is 1", actual1)
	})

	t.Run("it_should_get_the_first_from_cache", func(t *testing.T) {
		actual1 := *c.Get("A").value
		assert.Equal(t, "key A is 1", actual1)
	})

	t.Run("it_should_get_the_second_item", func(t *testing.T) {
		actual2 := *c.Get("B").value
		assert.Equal(t, "key B is 2", actual2)
	})

	t.Run("it_should_get_the_second_item_from_cache", func(t *testing.T) {
		actual2 := *c.Get("B").value
		assert.Equal(t, "key B is 2", actual2)
	})

	t.Run("it_should_get_the_first_item_from_cache_again", func(t *testing.T) {
		actual1 := *c.Get("A").value
		assert.Equal(t, "key A is 1", actual1)
	})

	// Expire the cache
	time.Sleep(ttl)

	t.Run("it_should_get_the_third_item", func(t *testing.T) {
		actual3 := *c.Get("A").value
		assert.Equal(t, "key A is 3", actual3)
	})

	t.Run("it_should_get_the_third_item_from_cache", func(t *testing.T) {
		actual3 := *c.Get("A").value
		assert.Equal(t, "key A is 3", actual3)
	})

	t.Run("it_should_return_error", func(t *testing.T) {
		actualErr := c.Get("B").err
		assert.EqualError(t, actualErr, "items depleted")
	})

}
