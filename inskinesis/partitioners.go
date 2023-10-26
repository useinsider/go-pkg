package inskinesis

import uuid "github.com/google/uuid"

// PartitionerFunction is the common signature of all partitioners, it maps a record to a partition key.
type PartitionerFunction func(record interface{}) string

type partitionersCollection struct{}

var Partitioners = partitionersCollection{}

// UUID partitioner returns a new uuid instead, regardless of parameters.
// Example: `newPartitionKey := Partitioners.UUID(nil)`
func (p *partitionersCollection) UUID(_ interface{}) string {
	return uuid.New().String()
}

func UUID(_ interface{}) string {
	return uuid.New().String()
}

// PartitionerPointer returns a pointer to the PartitionerPointer value passed in.
func PartitionerPointer(function PartitionerFunction) *PartitionerFunction {
	fn := &function
	return fn
}
