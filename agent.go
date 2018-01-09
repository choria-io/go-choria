package mcorpc

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// Agent is an instance of the MCollective compatible RPC agents
type Agent struct {
	Log    *logrus.Entry
	Config *choria.Config
	Choria *choria.Framework

	meta    *agents.Metadata
	actions map[string]func(*Request, *Reply, *Agent, choria.ConnectorInfo)
}

type AuditMessage struct {
	TimeStamp   string          `json:"timestamp"`
	RequestID   string          `json:"request_id"`
	RequestTime int64           `json:"request_time"`
	CallerID    string          `json:"caller"`
	Sender      string          `json:"sender"`
	Agent       string          `json:"agent"`
	Action      string          `json:"action"`
	Data        json.RawMessage `json:"data"`
}

// New creates a new MCollective SimpleRPC compatible agent
func New(name string, metadata *agents.Metadata, fw *choria.Framework, log *logrus.Entry) *Agent {
	a := &Agent{
		meta:    metadata,
		Log:     log.WithFields(logrus.Fields{"agent": name}),
		actions: make(map[string]func(*Request, *Reply, *Agent, choria.ConnectorInfo)),
		Choria:  fw,
		Config:  fw.Config,
	}

	return a
}

// RegisterAction registers an action into the agent
func (a *Agent) RegisterAction(name string, f func(*Request, *Reply, *Agent, choria.ConnectorInfo)) error {
	if _, ok := a.actions[name]; ok {
		return fmt.Errorf("Cannot register action %s, it already exist", name)
	}

	a.actions[name] = f

	return nil
}

// HandleMessage attempts to parse a choria.Message as a MCollective SimpleRPC request and calls
// the agents and actions associated with it
func (a *Agent) HandleMessage(msg *choria.Message, request protocol.Request, conn choria.ConnectorInfo, outbox chan *agents.AgentReply) {
	var err error

	reply := a.newReply()
	defer a.publish(reply, msg, request, outbox)

	rpcrequest, err := a.parseIncomingMessage(msg.Payload)
	if err != nil {
		reply.Statuscode = InvalidData
		reply.Statusmsg = fmt.Sprintf("Could not process request: %s", err.Error())
		return
	}

	action, found := a.actions[rpcrequest.Action]
	if !found {
		reply.Statuscode = UnknownAction
		reply.Statusmsg = fmt.Sprintf("Unknown action %s for agent %s", rpcrequest.Action, a.Name())
		return
	}

	// TODO:
	//  authorize
	//  timeouts

	if a.Config.RPCAudit {
		a.auditRequest(request, rpcrequest)
	}

	a.Log.Infof("Handling message %s for %s#%s from %s", msg.RequestID, a.Name(), rpcrequest.Action, request.CallerID())

	action(rpcrequest, reply, a, conn)
}

func (a *Agent) auditRequest(request protocol.Request, mcrequest *Request) {
	if !a.Config.RPCAudit {
		return
	}

	logfile := a.Config.Option("plugin.rpcaudit.logfile", "")

	if logfile == "" {
		a.Log.Warnf("MCollective RPC Auditing is enabled but no logfile is configured, skipping")
		return
	}

	amsg := AuditMessage{
		TimeStamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000000-0700"),
		RequestID:   request.RequestID(),
		RequestTime: request.Time().UTC().Unix(),
		CallerID:    request.CallerID(),
		Sender:      request.SenderID(),
		Agent:       mcrequest.Agent,
		Action:      mcrequest.Action,
		Data:        mcrequest.Data,
	}

	j, err := json.Marshal(amsg)
	if err != nil {
		a.Log.Warnf("Auditing is not functional because the auditing data could not be represented as JSON: %s", err)
		return
	}

	auditLock.Lock()
	defer auditLock.Unlock()

	f, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		a.Log.Warnf("Auditing is not functional because opening the logfile '%s' failed: %s", logfile, err)
		return
	}
	defer f.Close()

	_, err = f.Write(j)
	if err != nil {
		a.Log.Warnf("Auditing is not functional because writing to logfile '%s' failed: %s", logfile, err)
		return
	}
}

// Name retrieves the name of the agent
func (a *Agent) Name() string {
	return a.meta.Name
}

// Metadata retrieves the agent metadata
func (a *Agent) Metadata() *agents.Metadata {
	return a.meta
}

func (a *Agent) publish(rpcreply *Reply, msg *choria.Message, request protocol.Request, outbox chan *agents.AgentReply) {
	reply := &agents.AgentReply{
		Message: msg,
		Request: request,
	}

	j, err := json.Marshal(rpcreply)
	if err != nil {
		logrus.Errorf("Could not JSON encode reply: %s", err.Error())
		reply.Error = err
	}

	reply.Body = j

	outbox <- reply
}

func (a *Agent) newReply() *Reply {
	reply := &Reply{
		Statuscode: OK,
		Statusmsg:  "OK",
	}

	return reply
}

func (a *Agent) parseIncomingMessage(msg string) (*Request, error) {
	r := &Request{}

	err := json.Unmarshal([]byte(msg), r)
	if err != nil {
		return nil, fmt.Errorf("Could not parse incoming message as a MCollective SimpleRPC Request: %s", err.Error())
	}

	return r, nil
}
