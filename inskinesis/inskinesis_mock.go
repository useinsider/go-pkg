package inskinesis

import (
	reflect "reflect"

	kinesis "github.com/aws/aws-sdk-go/service/kinesis"
	gomock "go.uber.org/mock/gomock"
)

// MockKinesisInterface is a mock of KinesisInterface interface.
type MockKinesisInterface struct {
	ctrl     *gomock.Controller
	recorder *MockKinesisInterfaceMockRecorder
}

// MockKinesisInterfaceMockRecorder is the mock recorder for MockKinesisInterface.
type MockKinesisInterfaceMockRecorder struct {
	mock *MockKinesisInterface
}

// NewMockKinesisInterface creates a new mock instance.
func NewMockKinesisInterface(ctrl *gomock.Controller) *MockKinesisInterface {
	mock := &MockKinesisInterface{ctrl: ctrl}
	mock.recorder = &MockKinesisInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKinesisInterface) EXPECT() *MockKinesisInterfaceMockRecorder {
	return m.recorder
}

// PutRecords mocks base method.
func (m *MockKinesisInterface) PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutRecords", input)
	ret0, _ := ret[0].(*kinesis.PutRecordsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PutRecords indicates an expected call of PutRecords.
func (mr *MockKinesisInterfaceMockRecorder) PutRecords(input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutRecords", reflect.TypeOf((*MockKinesisInterface)(nil).PutRecords), input)
}

// MockStreamInterface is a mock of StreamInterface interface.
type MockStreamInterface struct {
	ctrl     *gomock.Controller
	recorder *MockStreamInterfaceMockRecorder
}

// MockStreamInterfaceMockRecorder is the mock recorder for MockStreamInterface.
type MockStreamInterfaceMockRecorder struct {
	mock *MockStreamInterface
}

// NewMockStreamInterface creates a new mock instance.
func NewMockStreamInterface(ctrl *gomock.Controller) *MockStreamInterface {
	mock := &MockStreamInterface{ctrl: ctrl}
	mock.recorder = &MockStreamInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamInterface) EXPECT() *MockStreamInterfaceMockRecorder {
	return m.recorder
}

// FlushAndStopStreaming mocks base method.
func (m *MockStreamInterface) FlushAndStopStreaming() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "FlushAndStopStreaming")
}

// FlushAndStopStreaming indicates an expected call of FlushAndStopStreaming.
func (mr *MockStreamInterfaceMockRecorder) FlushAndStopStreaming() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlushAndStopStreaming", reflect.TypeOf((*MockStreamInterface)(nil).FlushAndStopStreaming))
}

// Put mocks base method.
func (m *MockStreamInterface) Put(record any) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", record)
	ret0, _ := ret[0].(error)
	return ret0
}

// Put indicates an expected call of Put.
func (mr *MockStreamInterfaceMockRecorder) Put(record any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockStreamInterface)(nil).Put), record)
}
