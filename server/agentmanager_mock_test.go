// Code generated by MockGen. DO NOT EDIT.
// Source: agents/agents.go

// Package server is a generated GoMock package.
package server

import (
	context "context"
	json "encoding/json"
	reflect "reflect"
	time "time"

	aagent "github.com/choria-io/go-choria/aagent"
	build "github.com/choria-io/go-choria/build"
	config "github.com/choria-io/go-choria/config"
	inter "github.com/choria-io/go-choria/inter"
	lifecycle "github.com/choria-io/go-choria/lifecycle"
	protocol "github.com/choria-io/go-choria/protocol"
	ddl "github.com/choria-io/go-choria/providers/data/ddl"
	agents "github.com/choria-io/go-choria/server/agents"
	srvcache "github.com/choria-io/go-choria/srvcache"
	statistics "github.com/choria-io/go-choria/statistics"
	gomock "github.com/golang/mock/gomock"
)

// MockAgent is a mock of Agent interface.
type MockAgent struct {
	ctrl     *gomock.Controller
	recorder *MockAgentMockRecorder
}

// MockAgentMockRecorder is the mock recorder for MockAgent.
type MockAgentMockRecorder struct {
	mock *MockAgent
}

// NewMockAgent creates a new mock instance.
func NewMockAgent(ctrl *gomock.Controller) *MockAgent {
	mock := &MockAgent{ctrl: ctrl}
	mock.recorder = &MockAgentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAgent) EXPECT() *MockAgentMockRecorder {
	return m.recorder
}

// HandleMessage mocks base method.
func (m *MockAgent) HandleMessage(arg0 context.Context, arg1 inter.Message, arg2 protocol.Request, arg3 inter.ConnectorInfo, arg4 chan *agents.AgentReply) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleMessage", arg0, arg1, arg2, arg3, arg4)
}

// HandleMessage indicates an expected call of HandleMessage.
func (mr *MockAgentMockRecorder) HandleMessage(arg0, arg1, arg2, arg3, arg4 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleMessage", reflect.TypeOf((*MockAgent)(nil).HandleMessage), arg0, arg1, arg2, arg3, arg4)
}

// Metadata mocks base method.
func (m *MockAgent) Metadata() *agents.Metadata {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Metadata")
	ret0, _ := ret[0].(*agents.Metadata)
	return ret0
}

// Metadata indicates an expected call of Metadata.
func (mr *MockAgentMockRecorder) Metadata() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Metadata", reflect.TypeOf((*MockAgent)(nil).Metadata))
}

// Name mocks base method.
func (m *MockAgent) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockAgentMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockAgent)(nil).Name))
}

// ServerInfo mocks base method.
func (m *MockAgent) ServerInfo() agents.ServerInfoSource {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerInfo")
	ret0, _ := ret[0].(agents.ServerInfoSource)
	return ret0
}

// ServerInfo indicates an expected call of ServerInfo.
func (mr *MockAgentMockRecorder) ServerInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerInfo", reflect.TypeOf((*MockAgent)(nil).ServerInfo))
}

// SetServerInfo mocks base method.
func (m *MockAgent) SetServerInfo(arg0 agents.ServerInfoSource) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetServerInfo", arg0)
}

// SetServerInfo indicates an expected call of SetServerInfo.
func (mr *MockAgentMockRecorder) SetServerInfo(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetServerInfo", reflect.TypeOf((*MockAgent)(nil).SetServerInfo), arg0)
}

// ShouldActivate mocks base method.
func (m *MockAgent) ShouldActivate() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldActivate")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldActivate indicates an expected call of ShouldActivate.
func (mr *MockAgentMockRecorder) ShouldActivate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldActivate", reflect.TypeOf((*MockAgent)(nil).ShouldActivate))
}

// MockServerInfoSource is a mock of ServerInfoSource interface.
type MockServerInfoSource struct {
	ctrl     *gomock.Controller
	recorder *MockServerInfoSourceMockRecorder
}

// MockServerInfoSourceMockRecorder is the mock recorder for MockServerInfoSource.
type MockServerInfoSourceMockRecorder struct {
	mock *MockServerInfoSource
}

// NewMockServerInfoSource creates a new mock instance.
func NewMockServerInfoSource(ctrl *gomock.Controller) *MockServerInfoSource {
	mock := &MockServerInfoSource{ctrl: ctrl}
	mock.recorder = &MockServerInfoSourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServerInfoSource) EXPECT() *MockServerInfoSourceMockRecorder {
	return m.recorder
}

// AgentMetadata mocks base method.
func (m *MockServerInfoSource) AgentMetadata(arg0 string) (agents.Metadata, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AgentMetadata", arg0)
	ret0, _ := ret[0].(agents.Metadata)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// AgentMetadata indicates an expected call of AgentMetadata.
func (mr *MockServerInfoSourceMockRecorder) AgentMetadata(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentMetadata", reflect.TypeOf((*MockServerInfoSource)(nil).AgentMetadata), arg0)
}

// BuildInfo mocks base method.
func (m *MockServerInfoSource) BuildInfo() *build.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BuildInfo")
	ret0, _ := ret[0].(*build.Info)
	return ret0
}

// BuildInfo indicates an expected call of BuildInfo.
func (mr *MockServerInfoSourceMockRecorder) BuildInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BuildInfo", reflect.TypeOf((*MockServerInfoSource)(nil).BuildInfo))
}

// Classes mocks base method.
func (m *MockServerInfoSource) Classes() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Classes")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Classes indicates an expected call of Classes.
func (mr *MockServerInfoSourceMockRecorder) Classes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Classes", reflect.TypeOf((*MockServerInfoSource)(nil).Classes))
}

// ConfigFile mocks base method.
func (m *MockServerInfoSource) ConfigFile() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConfigFile")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConfigFile indicates an expected call of ConfigFile.
func (mr *MockServerInfoSourceMockRecorder) ConfigFile() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigFile", reflect.TypeOf((*MockServerInfoSource)(nil).ConfigFile))
}

// ConnectedServer mocks base method.
func (m *MockServerInfoSource) ConnectedServer() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectedServer")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConnectedServer indicates an expected call of ConnectedServer.
func (mr *MockServerInfoSourceMockRecorder) ConnectedServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedServer", reflect.TypeOf((*MockServerInfoSource)(nil).ConnectedServer))
}

// DataFuncMap mocks base method.
func (m *MockServerInfoSource) DataFuncMap() (ddl.FuncMap, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DataFuncMap")
	ret0, _ := ret[0].(ddl.FuncMap)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DataFuncMap indicates an expected call of DataFuncMap.
func (mr *MockServerInfoSourceMockRecorder) DataFuncMap() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DataFuncMap", reflect.TypeOf((*MockServerInfoSource)(nil).DataFuncMap))
}

// Facts mocks base method.
func (m *MockServerInfoSource) Facts() json.RawMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Facts")
	ret0, _ := ret[0].(json.RawMessage)
	return ret0
}

// Facts indicates an expected call of Facts.
func (mr *MockServerInfoSourceMockRecorder) Facts() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Facts", reflect.TypeOf((*MockServerInfoSource)(nil).Facts))
}

// Identity mocks base method.
func (m *MockServerInfoSource) Identity() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Identity")
	ret0, _ := ret[0].(string)
	return ret0
}

// Identity indicates an expected call of Identity.
func (mr *MockServerInfoSourceMockRecorder) Identity() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Identity", reflect.TypeOf((*MockServerInfoSource)(nil).Identity))
}

// KnownAgents mocks base method.
func (m *MockServerInfoSource) KnownAgents() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KnownAgents")
	ret0, _ := ret[0].([]string)
	return ret0
}

// KnownAgents indicates an expected call of KnownAgents.
func (mr *MockServerInfoSourceMockRecorder) KnownAgents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KnownAgents", reflect.TypeOf((*MockServerInfoSource)(nil).KnownAgents))
}

// LastProcessedMessage mocks base method.
func (m *MockServerInfoSource) LastProcessedMessage() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastProcessedMessage")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// LastProcessedMessage indicates an expected call of LastProcessedMessage.
func (mr *MockServerInfoSourceMockRecorder) LastProcessedMessage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastProcessedMessage", reflect.TypeOf((*MockServerInfoSource)(nil).LastProcessedMessage))
}

// MachineTransition mocks base method.
func (m *MockServerInfoSource) MachineTransition(name, version, path, id, transition string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MachineTransition", name, version, path, id, transition)
	ret0, _ := ret[0].(error)
	return ret0
}

// MachineTransition indicates an expected call of MachineTransition.
func (mr *MockServerInfoSourceMockRecorder) MachineTransition(name, version, path, id, transition any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MachineTransition", reflect.TypeOf((*MockServerInfoSource)(nil).MachineTransition), name, version, path, id, transition)
}

// MachinesStatus mocks base method.
func (m *MockServerInfoSource) MachinesStatus() ([]aagent.MachineState, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MachinesStatus")
	ret0, _ := ret[0].([]aagent.MachineState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MachinesStatus indicates an expected call of MachinesStatus.
func (mr *MockServerInfoSourceMockRecorder) MachinesStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MachinesStatus", reflect.TypeOf((*MockServerInfoSource)(nil).MachinesStatus))
}

// NewEvent mocks base method.
func (m *MockServerInfoSource) NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error {
	m.ctrl.T.Helper()
	varargs := []any{t}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewEvent", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// NewEvent indicates an expected call of NewEvent.
func (mr *MockServerInfoSourceMockRecorder) NewEvent(t any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{t}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewEvent", reflect.TypeOf((*MockServerInfoSource)(nil).NewEvent), varargs...)
}

// PrepareForShutdown mocks base method.
func (m *MockServerInfoSource) PrepareForShutdown() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PrepareForShutdown")
	ret0, _ := ret[0].(error)
	return ret0
}

// PrepareForShutdown indicates an expected call of PrepareForShutdown.
func (mr *MockServerInfoSourceMockRecorder) PrepareForShutdown() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareForShutdown", reflect.TypeOf((*MockServerInfoSource)(nil).PrepareForShutdown))
}

// Provisioning mocks base method.
func (m *MockServerInfoSource) Provisioning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Provisioning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Provisioning indicates an expected call of Provisioning.
func (mr *MockServerInfoSourceMockRecorder) Provisioning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Provisioning", reflect.TypeOf((*MockServerInfoSource)(nil).Provisioning))
}

// StartTime mocks base method.
func (m *MockServerInfoSource) StartTime() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// StartTime indicates an expected call of StartTime.
func (mr *MockServerInfoSourceMockRecorder) StartTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTime", reflect.TypeOf((*MockServerInfoSource)(nil).StartTime))
}

// Stats mocks base method.
func (m *MockServerInfoSource) Stats() statistics.ServerStats {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stats")
	ret0, _ := ret[0].(statistics.ServerStats)
	return ret0
}

// Stats indicates an expected call of Stats.
func (mr *MockServerInfoSourceMockRecorder) Stats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stats", reflect.TypeOf((*MockServerInfoSource)(nil).Stats))
}

// UpTime mocks base method.
func (m *MockServerInfoSource) UpTime() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpTime")
	ret0, _ := ret[0].(int64)
	return ret0
}

// UpTime indicates an expected call of UpTime.
func (mr *MockServerInfoSourceMockRecorder) UpTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpTime", reflect.TypeOf((*MockServerInfoSource)(nil).UpTime))
}

// MockChoriaFramework is a mock of ChoriaFramework interface.
type MockChoriaFramework struct {
	ctrl     *gomock.Controller
	recorder *MockChoriaFrameworkMockRecorder
}

// MockChoriaFrameworkMockRecorder is the mock recorder for MockChoriaFramework.
type MockChoriaFrameworkMockRecorder struct {
	mock *MockChoriaFramework
}

// NewMockChoriaFramework creates a new mock instance.
func NewMockChoriaFramework(ctrl *gomock.Controller) *MockChoriaFramework {
	mock := &MockChoriaFramework{ctrl: ctrl}
	mock.recorder = &MockChoriaFrameworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChoriaFramework) EXPECT() *MockChoriaFrameworkMockRecorder {
	return m.recorder
}

// Certname mocks base method.
func (m *MockChoriaFramework) Certname() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Certname")
	ret0, _ := ret[0].(string)
	return ret0
}

// Certname indicates an expected call of Certname.
func (mr *MockChoriaFrameworkMockRecorder) Certname() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Certname", reflect.TypeOf((*MockChoriaFramework)(nil).Certname))
}

// Configuration mocks base method.
func (m *MockChoriaFramework) Configuration() *config.Config {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Configuration")
	ret0, _ := ret[0].(*config.Config)
	return ret0
}

// Configuration indicates an expected call of Configuration.
func (mr *MockChoriaFrameworkMockRecorder) Configuration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Configuration", reflect.TypeOf((*MockChoriaFramework)(nil).Configuration))
}

// FacterCmd mocks base method.
func (m *MockChoriaFramework) FacterCmd() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FacterCmd")
	ret0, _ := ret[0].(string)
	return ret0
}

// FacterCmd indicates an expected call of FacterCmd.
func (mr *MockChoriaFrameworkMockRecorder) FacterCmd() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FacterCmd", reflect.TypeOf((*MockChoriaFramework)(nil).FacterCmd))
}

// FacterDomain mocks base method.
func (m *MockChoriaFramework) FacterDomain() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FacterDomain")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FacterDomain indicates an expected call of FacterDomain.
func (mr *MockChoriaFrameworkMockRecorder) FacterDomain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FacterDomain", reflect.TypeOf((*MockChoriaFramework)(nil).FacterDomain))
}

// MiddlewareServers mocks base method.
func (m *MockChoriaFramework) MiddlewareServers() (srvcache.Servers, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MiddlewareServers")
	ret0, _ := ret[0].(srvcache.Servers)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MiddlewareServers indicates an expected call of MiddlewareServers.
func (mr *MockChoriaFrameworkMockRecorder) MiddlewareServers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MiddlewareServers", reflect.TypeOf((*MockChoriaFramework)(nil).MiddlewareServers))
}

// NewTransportFromJSON mocks base method.
func (m *MockChoriaFramework) NewTransportFromJSON(data string) (protocol.TransportMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewTransportFromJSON", data)
	ret0, _ := ret[0].(protocol.TransportMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewTransportFromJSON indicates an expected call of NewTransportFromJSON.
func (mr *MockChoriaFrameworkMockRecorder) NewTransportFromJSON(data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewTransportFromJSON", reflect.TypeOf((*MockChoriaFramework)(nil).NewTransportFromJSON), data)
}

// ProvisionMode mocks base method.
func (m *MockChoriaFramework) ProvisionMode() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProvisionMode")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ProvisionMode indicates an expected call of ProvisionMode.
func (mr *MockChoriaFrameworkMockRecorder) ProvisionMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProvisionMode", reflect.TypeOf((*MockChoriaFramework)(nil).ProvisionMode))
}

// SupportsProvisioning mocks base method.
func (m *MockChoriaFramework) SupportsProvisioning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SupportsProvisioning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// SupportsProvisioning indicates an expected call of SupportsProvisioning.
func (mr *MockChoriaFrameworkMockRecorder) SupportsProvisioning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SupportsProvisioning", reflect.TypeOf((*MockChoriaFramework)(nil).SupportsProvisioning))
}

// UniqueID mocks base method.
func (m *MockChoriaFramework) UniqueID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UniqueID")
	ret0, _ := ret[0].(string)
	return ret0
}

// UniqueID indicates an expected call of UniqueID.
func (mr *MockChoriaFrameworkMockRecorder) UniqueID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UniqueID", reflect.TypeOf((*MockChoriaFramework)(nil).UniqueID))
}