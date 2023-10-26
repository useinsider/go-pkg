package inskinesis

import (
	kinesis "github.com/aws/aws-sdk-go/service/kinesis"
	reflect "reflect"

	protocol "github.com/aws/aws-sdk-go/private/protocol"
	eventstream "github.com/aws/aws-sdk-go/private/protocol/eventstream"
	gomock "go.uber.org/mock/gomock"
)

// MockSubscribeToShardEventStreamEvent is a mock of SubscribeToShardEventStreamEvent interface.
type MockSubscribeToShardEventStreamEvent struct {
	ctrl     *gomock.Controller
	recorder *MockSubscribeToShardEventStreamEventMockRecorder
}

// MockSubscribeToShardEventStreamEventMockRecorder is the mock recorder for MockSubscribeToShardEventStreamEvent.
type MockSubscribeToShardEventStreamEventMockRecorder struct {
	mock *MockSubscribeToShardEventStreamEvent
}

// NewMockSubscribeToShardEventStreamEvent creates a new mock instance.
func NewMockSubscribeToShardEventStreamEvent(ctrl *gomock.Controller) *MockSubscribeToShardEventStreamEvent {
	mock := &MockSubscribeToShardEventStreamEvent{ctrl: ctrl}
	mock.recorder = &MockSubscribeToShardEventStreamEventMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscribeToShardEventStreamEvent) EXPECT() *MockSubscribeToShardEventStreamEventMockRecorder {
	return m.recorder
}

// MarshalEvent mocks base method.
func (m *MockSubscribeToShardEventStreamEvent) MarshalEvent(arg0 protocol.PayloadMarshaler) (eventstream.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MarshalEvent", arg0)
	ret0, _ := ret[0].(eventstream.Message)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MarshalEvent indicates an expected call of MarshalEvent.
func (mr *MockSubscribeToShardEventStreamEventMockRecorder) MarshalEvent(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MarshalEvent", reflect.TypeOf((*MockSubscribeToShardEventStreamEvent)(nil).MarshalEvent), arg0)
}

// UnmarshalEvent mocks base method.
func (m *MockSubscribeToShardEventStreamEvent) UnmarshalEvent(arg0 protocol.PayloadUnmarshaler, arg1 eventstream.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnmarshalEvent", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnmarshalEvent indicates an expected call of UnmarshalEvent.
func (mr *MockSubscribeToShardEventStreamEventMockRecorder) UnmarshalEvent(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnmarshalEvent", reflect.TypeOf((*MockSubscribeToShardEventStreamEvent)(nil).UnmarshalEvent), arg0, arg1)
}

// eventSubscribeToShardEventStream mocks base method.
func (m *MockSubscribeToShardEventStreamEvent) eventSubscribeToShardEventStream() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "eventSubscribeToShardEventStream")
}

// eventSubscribeToShardEventStream indicates an expected call of eventSubscribeToShardEventStream.
func (mr *MockSubscribeToShardEventStreamEventMockRecorder) eventSubscribeToShardEventStream() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "eventSubscribeToShardEventStream", reflect.TypeOf((*MockSubscribeToShardEventStreamEvent)(nil).eventSubscribeToShardEventStream))
}

// MockSubscribeToShardEventStreamReader is a mock of SubscribeToShardEventStreamReader interface.
type MockSubscribeToShardEventStreamReader struct {
	ctrl     *gomock.Controller
	recorder *MockSubscribeToShardEventStreamReaderMockRecorder
}

// MockSubscribeToShardEventStreamReaderMockRecorder is the mock recorder for MockSubscribeToShardEventStreamReader.
type MockSubscribeToShardEventStreamReaderMockRecorder struct {
	mock *MockSubscribeToShardEventStreamReader
}

// NewMockSubscribeToShardEventStreamReader creates a new mock instance.
func NewMockSubscribeToShardEventStreamReader(ctrl *gomock.Controller) *MockSubscribeToShardEventStreamReader {
	mock := &MockSubscribeToShardEventStreamReader{ctrl: ctrl}
	mock.recorder = &MockSubscribeToShardEventStreamReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscribeToShardEventStreamReader) EXPECT() *MockSubscribeToShardEventStreamReaderMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockSubscribeToShardEventStreamReader) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockSubscribeToShardEventStreamReaderMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSubscribeToShardEventStreamReader)(nil).Close))
}

// Err mocks base method.
func (m *MockSubscribeToShardEventStreamReader) Err() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Err")
	ret0, _ := ret[0].(error)
	return ret0
}

// Err indicates an expected call of Err.
func (mr *MockSubscribeToShardEventStreamReaderMockRecorder) Err() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Err", reflect.TypeOf((*MockSubscribeToShardEventStreamReader)(nil).Err))
}

// Events mocks base method.
func (m *MockSubscribeToShardEventStreamReader) Events() <-chan kinesis.SubscribeToShardEventStreamEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Events")
	ret0, _ := ret[0].(<-chan kinesis.SubscribeToShardEventStreamEvent)
	return ret0
}

// Events indicates an expected call of Events.
func (mr *MockSubscribeToShardEventStreamReaderMockRecorder) Events() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Events", reflect.TypeOf((*MockSubscribeToShardEventStreamReader)(nil).Events))
}
