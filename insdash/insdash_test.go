package insdash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const kb64 = 64 * 1024

func Test_createBatches(t *testing.T) {
	t.Run("should_return_empty_batches_when_empty", func(t *testing.T) {
		batches, err := CreateBatches([]string{}, 10, kb64)

		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, 0, len(batches), "batches length should be equal to 0")
	})

	t.Run("should_return_batches_when_not_empty", func(t *testing.T) {
		batches, err := CreateBatches([]string{"test"}, 10, kb64)

		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, 1, len(batches), "batches length should be equal to 1")
	})

	t.Run("should_return_batches_by_record_limit", func(t *testing.T) {
		batches, err := CreateBatches([]string{"test", "test"}, 1, kb64)

		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, 2, len(batches), "batches length should be equal to 2")
	})

	t.Run("should_return_batches_by_byte_limit", func(t *testing.T) {
		batches, err := CreateBatches([]string{createString(kb64), createString(kb64)}, 10, kb64)

		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, 2, len(batches), "batches length should be equal to 2")
	})
}

func createString(byteSize int) string {
	var str string
	for i := 0; i < byteSize; i++ {
		str += "a"
	}
	return str
}

func TestMapToValueSlice(t *testing.T) {
	t.Run("should_return_empty_slice_when_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]string{})
		assert.Equal(t, 0, len(s), "slice length should be equal to 0")
		assert.Equal(t, []string{}, s, "slice value should be equal to []string{}")
	})

	t.Run("should_return_string_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]string{"test": "test"})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, "test", s[0], "slice value should be equal to \"test\"")
	})

	t.Run("should_return_int_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]int{"test": 1})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, 1, s[0], "slice value should be equal to 1")
	})

	t.Run("should_return_bool_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]bool{"test": true})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, true, s[0], "slice value should be equal to true")
	})

	t.Run("should_return_float_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]float64{"test": 1.0})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, 1.0, s[0], "slice value should be equal to 1.0")
	})

	t.Run("should_return_struct_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]struct{}{"test": {}})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, struct{}{}, s[0], "slice value should be equal to struct{}{}")
	})

	t.Run("should_return_interface_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]interface{}{"test": 1})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, 1, s[0], "slice value should be equal to 1")
	})

	t.Run("should_return_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string][]string{"test": {"test"}})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, []string{"test"}, s[0], "slice value should be equal to []string{\"test\"}")
	})

	t.Run("should_return_map_slice_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]map[string]string{"test": {"test": "test"}})
		assert.Equal(t, 1, len(s), "slice length should be equal to 1")
		assert.Equal(t, map[string]string{"test": "test"}, s[0], "slice value should be equal to map[string]string{\"test\": \"test\"}")
	})

	t.Run("should_return_slice_with_multiple_values_when_not_empty", func(t *testing.T) {
		s := MapToValueSlice(map[string]string{"test": "test", "test2": "test2"})
		assert.Equal(t, 2, len(s), "slice length should be equal to 2")
		assert.Equal(t, []string{"test", "test2"}, s, "slice value should be equal to []string{\"test\", \"test2\"}")
	})
}
