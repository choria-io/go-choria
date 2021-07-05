// Code generated by MockGen. DO NOT EDIT.
// Source: conn.go

// Package spool is a generated GoMock package.
package submission

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	nats "github.com/nats-io/nats.go"
)

// MockConnector is a mock of Connector interface.
type MockConnector struct {
	ctrl     *gomock.Controller
	recorder *MockConnectorMockRecorder
}

// MockConnectorMockRecorder is the mock recorder for MockConnector.
type MockConnectorMockRecorder struct {
	mock *MockConnector
}

// NewMockConnector creates a new mock instance.
func NewMockConnector(ctrl *gomock.Controller) *MockConnector {
	mock := &MockConnector{ctrl: ctrl}
	mock.recorder = &MockConnectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnector) EXPECT() *MockConnectorMockRecorder {
	return m.recorder
}

// PublishRawMsg mocks base method.
func (m *MockConnector) PublishRawMsg(msg *nats.Msg) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishRawMsg", msg)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishRawMsg indicates an expected call of PublishRawMsg.
func (mr *MockConnectorMockRecorder) PublishRawMsg(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishRawMsg", reflect.TypeOf((*MockConnector)(nil).PublishRawMsg), msg)
}

// RequestRawMsgWithContext mocks base method.
func (m *MockConnector) RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestRawMsgWithContext", ctx, msg)
	ret0, _ := ret[0].(*nats.Msg)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RequestRawMsgWithContext indicates an expected call of RequestRawMsgWithContext.
func (mr *MockConnectorMockRecorder) RequestRawMsgWithContext(ctx, msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestRawMsgWithContext", reflect.TypeOf((*MockConnector)(nil).RequestRawMsgWithContext), ctx, msg)
}