// Code generated by MockGen. DO NOT EDIT.
// Source: ../connector.go

// Package imock is a generated GoMock package.
package imock

import (
	context "context"
	reflect "reflect"

	inter "github.com/choria-io/go-choria/inter"
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

// AgentBroadcastTarget mocks base method.
func (m *MockConnector) AgentBroadcastTarget(collective, agent string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AgentBroadcastTarget", collective, agent)
	ret0, _ := ret[0].(string)
	return ret0
}

// AgentBroadcastTarget indicates an expected call of AgentBroadcastTarget.
func (mr *MockConnectorMockRecorder) AgentBroadcastTarget(collective, agent interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentBroadcastTarget", reflect.TypeOf((*MockConnector)(nil).AgentBroadcastTarget), collective, agent)
}

// ChanQueueSubscribe mocks base method.
func (m *MockConnector) ChanQueueSubscribe(name, subject, group string, capacity int) (chan inter.ConnectorMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChanQueueSubscribe", name, subject, group, capacity)
	ret0, _ := ret[0].(chan inter.ConnectorMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ChanQueueSubscribe indicates an expected call of ChanQueueSubscribe.
func (mr *MockConnectorMockRecorder) ChanQueueSubscribe(name, subject, group, capacity interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChanQueueSubscribe", reflect.TypeOf((*MockConnector)(nil).ChanQueueSubscribe), name, subject, group, capacity)
}

// Close mocks base method.
func (m *MockConnector) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockConnectorMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockConnector)(nil).Close))
}

// Connect mocks base method.
func (m *MockConnector) Connect(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockConnectorMockRecorder) Connect(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockConnector)(nil).Connect), ctx)
}

// ConnectedServer mocks base method.
func (m *MockConnector) ConnectedServer() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectedServer")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConnectedServer indicates an expected call of ConnectedServer.
func (mr *MockConnectorMockRecorder) ConnectedServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedServer", reflect.TypeOf((*MockConnector)(nil).ConnectedServer))
}

// ConnectionOptions mocks base method.
func (m *MockConnector) ConnectionOptions() nats.Options {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectionOptions")
	ret0, _ := ret[0].(nats.Options)
	return ret0
}

// ConnectionOptions indicates an expected call of ConnectionOptions.
func (mr *MockConnectorMockRecorder) ConnectionOptions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectionOptions", reflect.TypeOf((*MockConnector)(nil).ConnectionOptions))
}

// ConnectionStats mocks base method.
func (m *MockConnector) ConnectionStats() nats.Statistics {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectionStats")
	ret0, _ := ret[0].(nats.Statistics)
	return ret0
}

// ConnectionStats indicates an expected call of ConnectionStats.
func (mr *MockConnectorMockRecorder) ConnectionStats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectionStats", reflect.TypeOf((*MockConnector)(nil).ConnectionStats))
}

// InboxPrefix mocks base method.
func (m *MockConnector) InboxPrefix() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InboxPrefix")
	ret0, _ := ret[0].(string)
	return ret0
}

// InboxPrefix indicates an expected call of InboxPrefix.
func (mr *MockConnectorMockRecorder) InboxPrefix() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InboxPrefix", reflect.TypeOf((*MockConnector)(nil).InboxPrefix))
}

// IsConnected mocks base method.
func (m *MockConnector) IsConnected() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsConnected")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsConnected indicates an expected call of IsConnected.
func (mr *MockConnectorMockRecorder) IsConnected() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsConnected", reflect.TypeOf((*MockConnector)(nil).IsConnected))
}

// Nats mocks base method.
func (m *MockConnector) Nats() *nats.Conn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Nats")
	ret0, _ := ret[0].(*nats.Conn)
	return ret0
}

// Nats indicates an expected call of Nats.
func (mr *MockConnectorMockRecorder) Nats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nats", reflect.TypeOf((*MockConnector)(nil).Nats))
}

// NodeDirectedTarget mocks base method.
func (m *MockConnector) NodeDirectedTarget(collective, identity string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NodeDirectedTarget", collective, identity)
	ret0, _ := ret[0].(string)
	return ret0
}

// NodeDirectedTarget indicates an expected call of NodeDirectedTarget.
func (mr *MockConnectorMockRecorder) NodeDirectedTarget(collective, identity interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NodeDirectedTarget", reflect.TypeOf((*MockConnector)(nil).NodeDirectedTarget), collective, identity)
}

// Publish mocks base method.
func (m *MockConnector) Publish(msg inter.Message) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", msg)
	ret0, _ := ret[0].(error)
	return ret0
}

// Publish indicates an expected call of Publish.
func (mr *MockConnectorMockRecorder) Publish(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockConnector)(nil).Publish), msg)
}

// PublishRaw mocks base method.
func (m *MockConnector) PublishRaw(target string, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishRaw", target, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishRaw indicates an expected call of PublishRaw.
func (mr *MockConnectorMockRecorder) PublishRaw(target, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishRaw", reflect.TypeOf((*MockConnector)(nil).PublishRaw), target, data)
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

// QueueSubscribe mocks base method.
func (m *MockConnector) QueueSubscribe(ctx context.Context, name, subject, group string, output chan inter.ConnectorMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueSubscribe", ctx, name, subject, group, output)
	ret0, _ := ret[0].(error)
	return ret0
}

// QueueSubscribe indicates an expected call of QueueSubscribe.
func (mr *MockConnectorMockRecorder) QueueSubscribe(ctx, name, subject, group, output interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueSubscribe", reflect.TypeOf((*MockConnector)(nil).QueueSubscribe), ctx, name, subject, group, output)
}

// ReplyTarget mocks base method.
func (m *MockConnector) ReplyTarget(msg inter.Message) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReplyTarget", msg)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReplyTarget indicates an expected call of ReplyTarget.
func (mr *MockConnectorMockRecorder) ReplyTarget(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReplyTarget", reflect.TypeOf((*MockConnector)(nil).ReplyTarget), msg)
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

// ServiceBroadcastTarget mocks base method.
func (m *MockConnector) ServiceBroadcastTarget(collective, agent string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServiceBroadcastTarget", collective, agent)
	ret0, _ := ret[0].(string)
	return ret0
}

// ServiceBroadcastTarget indicates an expected call of ServiceBroadcastTarget.
func (mr *MockConnectorMockRecorder) ServiceBroadcastTarget(collective, agent interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServiceBroadcastTarget", reflect.TypeOf((*MockConnector)(nil).ServiceBroadcastTarget), collective, agent)
}

// Unsubscribe mocks base method.
func (m *MockConnector) Unsubscribe(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Unsubscribe", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unsubscribe indicates an expected call of Unsubscribe.
func (mr *MockConnectorMockRecorder) Unsubscribe(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockConnector)(nil).Unsubscribe), name)
}