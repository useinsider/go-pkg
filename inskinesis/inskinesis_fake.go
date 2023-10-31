package inskinesis

import (
	"encoding/json"
	"fmt"
)

type FakeStream struct {
	StreamInterface
	Data        []string
	Stream      StreamInterface
	Partitioner *PartitionerFunction
}

func (s *FakeStream) Put(v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		println(fmt.Sprintf("Error marshalling in fake kinesis %v", v))
	}
	s.Data = append(s.Data, string(js))
	if s.Stream != nil {
		s.Stream.Put(v)
	}
}

func (s *FakeStream) Get() {}

// Datum gets item at the index i, and converts JSON at the index i to interface pointer r.
// Returns JSON string.
// Example:
//
//	t := MyStruct{}
//	js := s.Datum(-1, &t)
func (s *FakeStream) Datum(i int, r interface{}) string {
	if i < 0 {
		i = len(s.Data) + i
	}
	d := s.Data[i]
	if r != nil {
		_ = json.Unmarshal([]byte(d), &r)
	}
	return d
}
