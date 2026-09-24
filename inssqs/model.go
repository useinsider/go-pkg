package inssqs

import "github.com/aws/aws-sdk-go-v2/service/sqs/types"

type entry interface {
	getID() *string
}

type SQSMessageEntry struct {
	Id                     *string
	MessageBody            *string
	DelaySeconds           int32
	MessageDeduplicationId *string
	MessageGroupId         *string
}

func (e SQSMessageEntry) getID() *string {
	return e.Id
}

func (e SQSMessageEntry) toSendMessageBatchRequestEntry() types.SendMessageBatchRequestEntry {
	return types.SendMessageBatchRequestEntry{
		Id:                     e.Id,
		MessageBody:            e.MessageBody,
		DelaySeconds:           e.DelaySeconds,
		MessageDeduplicationId: e.MessageDeduplicationId,
		MessageGroupId:         e.MessageGroupId,
	}
}

type SQSDeleteMessageEntry struct {
	Id            *string
	ReceiptHandle *string
}

func (e SQSDeleteMessageEntry) getID() *string {
	return e.Id
}

func (e SQSDeleteMessageEntry) toDeleteMessageBatchRequestEntry() types.DeleteMessageBatchRequestEntry {
	return types.DeleteMessageBatchRequestEntry{
		Id:            e.Id,
		ReceiptHandle: e.ReceiptHandle,
	}
}
