package insdash

import "encoding/json"

func CreateBatches[T any](v []T, recordLimit int, byteLimit int) ([][]T, error) {
	batches := make([][]T, 0)
	buffer := make([]T, 0)
	bufferSize := 0

	for _, record := range v {
		js, jsErr := json.Marshal(record)
		if jsErr != nil {
			return nil, jsErr
		}

		recordSize := len(js)
		sizeExceeds := bufferSize+recordSize > byteLimit
		bufferFull := len(buffer) == recordLimit

		if len(buffer) > 0 && (bufferFull || sizeExceeds) {
			batches = append(batches, buffer)
			buffer = make([]T, 0)
			bufferSize = 0
		}

		buffer = append(buffer, record)
		bufferSize += recordSize
	}

	if len(buffer) > 0 {
		batches = append(batches, buffer)
	}

	return batches, nil
}

func Contains[T comparable](v []T, e T) bool {
	for _, a := range v {
		if a == e {
			return true
		}
	}
	return false
}

func MapToValueSlice[K comparable, V any](m map[K]V) []V {
	v := make([]V, 0, len(m))
	for _, value := range m {
		v = append(v, value)
	}
	return v
}
