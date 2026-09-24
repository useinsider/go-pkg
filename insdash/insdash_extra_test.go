package insdash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	t.Run("should_return_true_when_element_present", func(t *testing.T) {
		assert.True(t, Contains([]string{"a", "b", "c"}, "b"))
	})

	t.Run("should_return_false_when_element_absent", func(t *testing.T) {
		assert.False(t, Contains([]string{"a", "b", "c"}, "z"))
	})

	t.Run("should_return_false_for_empty_slice", func(t *testing.T) {
		assert.False(t, Contains([]int{}, 1))
	})

	t.Run("should_work_with_comparable_non_string_types", func(t *testing.T) {
		assert.True(t, Contains([]int{1, 2, 3}, 3))
	})
}

func TestCreateBatches_MarshalError(t *testing.T) {
	t.Run("should_return_error_for_unmarshalable_record", func(t *testing.T) {
		batches, err := CreateBatches([]chan int{make(chan int)}, 10, 1024)

		assert.Error(t, err, "channels cannot be JSON-marshaled")
		assert.Nil(t, batches)
	})
}
