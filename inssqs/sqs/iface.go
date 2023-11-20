package sqs

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type API interface {
	SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error)
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	DeleteMessageBatch(ctx context.Context, params *sqs.DeleteMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error)
}

type proxy struct {
	client *sqs.Client
}

func NewSQSProxy(client *sqs.Client) API {
	return &proxy{client}
}

func (p *proxy) SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error) {
	return p.client.SendMessageBatch(ctx, params, optFns...)
}

func (p *proxy) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	return p.client.GetQueueUrl(ctx, params, optFns...)
}

func (p *proxy) DeleteMessageBatch(ctx context.Context, params *sqs.DeleteMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error) {
	return p.client.DeleteMessageBatch(ctx, params, optFns...)
}
