package sqs

import (
	context "context"
	reflect "reflect"

	sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	gomock "go.uber.org/mock/gomock"
)

// MockAPI is a mock of API interface.
type MockAPI struct {
	ctrl     *gomock.Controller
	recorder *MockAPIMockRecorder
}

// MockAPIMockRecorder is the mock recorder for MockAPI.
type MockAPIMockRecorder struct {
	mock *MockAPI
}

// NewMockSQS creates a new mock instance.
func NewMockSQS(ctrl *gomock.Controller) *MockAPI {
	mock := &MockAPI{ctrl: ctrl}
	mock.recorder = &MockAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAPI) EXPECT() *MockAPIMockRecorder {
	return m.recorder
}

// GetQueueUrl mocks base method.
func (m *MockAPI) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetQueueUrl", varargs...)
	ret0, _ := ret[0].(*sqs.GetQueueUrlOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetQueueUrl indicates an expected call of GetQueueUrl.
func (mr *MockAPIMockRecorder) GetQueueUrl(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetQueueUrl", reflect.TypeOf((*MockAPI)(nil).GetQueueUrl), varargs...)
}

// SendMessageBatch mocks base method.
func (m *MockAPI) SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SendMessageBatch", varargs...)
	ret0, _ := ret[0].(*sqs.SendMessageBatchOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendMessageBatch indicates an expected call of SendMessageBatch.
func (mr *MockAPIMockRecorder) SendMessageBatch(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessageBatch", reflect.TypeOf((*MockAPI)(nil).SendMessageBatch), varargs...)
}

// DeleteMessageBatch mocks base method.
func (m *MockAPI) DeleteMessageBatch(ctx context.Context, params *sqs.DeleteMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, params}
	for _, a := range optFns {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteMessageBatch", varargs...)
	ret0, _ := ret[0].(*sqs.DeleteMessageBatchOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteMessageBatch indicates an expected call of DeleteMessageBatch.
func (mr *MockAPIMockRecorder) DeleteMessageBatch(ctx, params any, optFns ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, params}, optFns...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMessageBatch", reflect.TypeOf((*MockAPI)(nil).DeleteMessageBatch), varargs...)
}
