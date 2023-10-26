package inskinesis

import (
	"encoding/json"
	"errors"
	"reflect"
)

func TakeSliceArg(arg interface{}) (out []interface{}, ok bool) {
	slice, success := takeArg(arg, reflect.Slice)
	if !success {
		ok = false
		return
	}
	c := slice.Len()
	out = make([]interface{}, c)
	for i := 0; i < c; i++ {
		out[i] = slice.Index(i).Interface()
	}
	return out, true
}

func takeArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

func createBatches(v interface{}, recordLimit int, byteLimit int) ([][]interface{}, error) {
	records, ok := TakeSliceArg(v)
	if !ok {
		return nil, errors.New("invalid input")
	}

	var batches = make([][]interface{}, 0)
	buffer := make([]interface{}, 0)
	bufferSize := 0

	for _, record := range records {
		js, jsErr := json.Marshal(record)
		if jsErr != nil {
			return nil, jsErr
		}

		recordSize := len(js)
		sizeExceeds := bufferSize+recordSize > byteLimit
		bufferFull := len(buffer) == recordLimit

		if len(buffer) > 0 && (bufferFull || sizeExceeds) {
			batches = append(batches, buffer)
			buffer = make([]interface{}, 0)
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
