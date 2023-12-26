package inssqs

type FakeQueue struct {
	Interface
	Data []SQSMessageEntry
}

func (q *FakeQueue) SendMessageBatch(entries []SQSMessageEntry) (failed []SQSMessageEntry, err error) {
	q.Data = append(q.Data, entries...)
	return nil, nil
}

func (q *FakeQueue) DeleteMessageBatch(entries []SQSDeleteMessageEntry) (failed []SQSDeleteMessageEntry, err error) {
	newData := make([]SQSMessageEntry, 0)
	for _, e := range q.Data {
		for _, de := range entries {
			if e.Id != de.Id {
				newData = append(newData, e)
			}
		}
	}

	q.Data = newData
	return nil, nil
}
