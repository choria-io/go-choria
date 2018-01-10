package mcorpc

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc/audit"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// Action is a function that implements a RPC Action
type Action func(*Request, *Reply, *Agent, choria.ConnectorInfo)

// Agent is an instance of the MCollective compatible RPC agents
type Agent struct {
	Log    *logrus.Entry
	Config *choria.Config
	Choria *choria.Framework

	meta    *agents.Metadata
	actions map[string]Action
}

// New creates a new MCollective SimpleRPC compatible agent
func New(name string, metadata *agents.Metadata, fw *choria.Framework, log *logrus.Entry) *Agent {
	a := &Agent{
		meta:    metadata,
		Log:     log.WithFields(logrus.Fields{"agent": name}),
		actions: make(map[string]Action),
		Choria:  fw,
		Config:  fw.Config,
	}

	return a
}

// RegisterAction registers an action into the agent
func (a *Agent) RegisterAction(name string, f Action) error {
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
		audit.Request(request, rpcrequest.Agent, rpcrequest.Action, rpcrequest.Data, a.Config)
	}

	a.Log.Infof("Handling message %s for %s#%s from %s", msg.RequestID, a.Name(), rpcrequest.Action, request.CallerID())

	action(rpcrequest, reply, a, conn)
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
