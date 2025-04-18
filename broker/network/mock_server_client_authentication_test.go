// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/nats-io/nats-server/v2/server (interfaces: ClientAuthentication)

// mockgen -package network github.com/nats-io/nats-server/v2/server ClientAuthentication

// Package network is a generated GoMock package.
package network

import (
	tls "crypto/tls"
	net "net"
	reflect "reflect"

	server "github.com/nats-io/nats-server/v2/server"
	gomock "go.uber.org/mock/gomock"
)

// MockClientAuthentication is a mock of ClientAuthentication interface.
type MockClientAuthentication struct {
	ctrl     *gomock.Controller
	recorder *MockClientAuthenticationMockRecorder
}

// MockClientAuthenticationMockRecorder is the mock recorder for MockClientAuthentication.
type MockClientAuthenticationMockRecorder struct {
	mock *MockClientAuthentication
}

// NewMockClientAuthentication creates a new mock instance.
func NewMockClientAuthentication(ctrl *gomock.Controller) *MockClientAuthentication {
	mock := &MockClientAuthentication{ctrl: ctrl}
	mock.recorder = &MockClientAuthenticationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientAuthentication) EXPECT() *MockClientAuthenticationMockRecorder {
	return m.recorder
}

// GetNonce mocks base method.
func (m *MockClientAuthentication) GetNonce() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNonce")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetNonce indicates an expected call of GetNonce.
func (mr *MockClientAuthenticationMockRecorder) GetNonce() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNonce", reflect.TypeOf((*MockClientAuthentication)(nil).GetNonce))
}

// GetOpts mocks base method.
func (m *MockClientAuthentication) GetOpts() *server.ClientOpts {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOpts")
	ret0, _ := ret[0].(*server.ClientOpts)
	return ret0
}

// GetOpts indicates an expected call of GetOpts.
func (mr *MockClientAuthenticationMockRecorder) GetOpts() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOpts", reflect.TypeOf((*MockClientAuthentication)(nil).GetOpts))
}

// GetTLSConnectionState mocks base method.
func (m *MockClientAuthentication) GetTLSConnectionState() *tls.ConnectionState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTLSConnectionState")
	ret0, _ := ret[0].(*tls.ConnectionState)
	return ret0
}

// GetTLSConnectionState indicates an expected call of GetTLSConnectionState.
func (mr *MockClientAuthenticationMockRecorder) GetTLSConnectionState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTLSConnectionState", reflect.TypeOf((*MockClientAuthentication)(nil).GetTLSConnectionState))
}

// Kind mocks base method.
func (m *MockClientAuthentication) Kind() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Kind")
	ret0, _ := ret[0].(int)
	return ret0
}

// Kind indicates an expected call of Kind.
func (mr *MockClientAuthenticationMockRecorder) Kind() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kind", reflect.TypeOf((*MockClientAuthentication)(nil).Kind))
}

// RegisterUser mocks base method.
func (m *MockClientAuthentication) RegisterUser(arg0 *server.User) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterUser", arg0)
}

// RegisterUser indicates an expected call of RegisterUser.
func (mr *MockClientAuthenticationMockRecorder) RegisterUser(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterUser", reflect.TypeOf((*MockClientAuthentication)(nil).RegisterUser), arg0)
}

// RemoteAddress mocks base method.
func (m *MockClientAuthentication) RemoteAddress() net.Addr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoteAddress")
	ret0, _ := ret[0].(net.Addr)
	return ret0
}

// RemoteAddress indicates an expected call of RemoteAddress.
func (mr *MockClientAuthenticationMockRecorder) RemoteAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoteAddress", reflect.TypeOf((*MockClientAuthentication)(nil).RemoteAddress))
}
