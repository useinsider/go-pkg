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
		batches, err := CreateBatches([]string{"test", "test"}, 10, kb64)

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
