// Code generated by MockGen. DO NOT EDIT.
// Source: ../connector_message.go
//
// Generated by this command:
//
//	mockgen -write_generate_directive -destination connector_message.go -package imock -source ../connector_message.go
//

// Package imock is a generated GoMock package.
package imock

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -write_generate_directive -destination connector_message.go -package imock -source ../connector_message.go

// MockConnectorMessage is a mock of ConnectorMessage interface.
type MockConnectorMessage struct {
	ctrl     *gomock.Controller
	recorder *MockConnectorMessageMockRecorder
	isgomock struct{}
}

// MockConnectorMessageMockRecorder is the mock recorder for MockConnectorMessage.
type MockConnectorMessageMockRecorder struct {
	mock *MockConnectorMessage
}

// NewMockConnectorMessage creates a new mock instance.
func NewMockConnectorMessage(ctrl *gomock.Controller) *MockConnectorMessage {
	mock := &MockConnectorMessage{ctrl: ctrl}
	mock.recorder = &MockConnectorMessageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnectorMessage) EXPECT() *MockConnectorMessageMockRecorder {
	return m.recorder
}

// Data mocks base method.
func (m *MockConnectorMessage) Data() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Data")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Data indicates an expected call of Data.
func (mr *MockConnectorMessageMockRecorder) Data() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Data", reflect.TypeOf((*MockConnectorMessage)(nil).Data))
}

// Msg mocks base method.
func (m *MockConnectorMessage) Msg() any {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Msg")
	ret0, _ := ret[0].(any)
	return ret0
}

// Msg indicates an expected call of Msg.
func (mr *MockConnectorMessageMockRecorder) Msg() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Msg", reflect.TypeOf((*MockConnectorMessage)(nil).Msg))
}

// Reply mocks base method.
func (m *MockConnectorMessage) Reply() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reply")
	ret0, _ := ret[0].(string)
	return ret0
}

// Reply indicates an expected call of Reply.
func (mr *MockConnectorMessageMockRecorder) Reply() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reply", reflect.TypeOf((*MockConnectorMessage)(nil).Reply))
}

// Subject mocks base method.
func (m *MockConnectorMessage) Subject() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subject")
	ret0, _ := ret[0].(string)
	return ret0
}

// Subject indicates an expected call of Subject.
func (mr *MockConnectorMessageMockRecorder) Subject() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subject", reflect.TypeOf((*MockConnectorMessage)(nil).Subject))
}
