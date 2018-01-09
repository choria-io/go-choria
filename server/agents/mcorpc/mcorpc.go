package mcorpc

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// StatusCode is a reply status as defined by MCollective SimpleRPC - integers 0 to 5
//
// See the constants OK, RPCAborted, UnknownRPCAction, MissingRPCData, InvalidRPCData and UnknownRPCError
type StatusCode uint8

const (
	// OK is the reply status when all worked
	OK = StatusCode(iota)

	// Aborted is status for when the action could not run, most failures in an action should set this
	Aborted

	// UnknownAction is the status for unknown actions requested
	UnknownAction

	// MissingData is the status for missing input data
	MissingData

	// InvalidData is the status for invalid input data
	InvalidData

	// UnknownError is the status general failures in agents should set when things go bad
	UnknownError
)

// Agent is an instance of the MCollective compatible RPC agents
type Agent struct {
	Log    *logrus.Entry
	Config *choria.Config
	Choria *choria.Framework

	meta    *agents.Metadata
	actions map[string]func(*Request, *Reply, *Agent, choria.ConnectorInfo)
}

// Reply is the reply data as stipulated by MCollective RPC system.  The Data
// has to be something that can be turned into JSON using the normal Marshal system
type Reply struct {
	Statuscode StatusCode  `json:"statuscode"`
	Statusmsg  string      `json:"statusmsg"`
	Data       interface{} `json:"data"`
}

// Request is a request as defined by the MCollective RPC system
// NOTE: input arguments not yet handled
type Request struct {
	Agent  string          `json:"agent"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
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
	//  audit
	//  timeouts

	a.Log.Infof("Handling message %s for %s#%s from %s", msg.RequestID, a.Name(), rpcrequest.Action, request.CallerID())
	a.Log.Debugf("%#v", string(rpcrequest.Data))

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

// ParseRequestData parses the request parameters received from the client into a target structure
//
// Example used in a action:
//
//   var rparams struct {
//      Package string `json:"package"`
//   }
//
//   if !mcorpc.ParseRequestData(&rparams, req, reply) {
//     // the function already set appropriate errors on reply
//	   return
//   }
//
//   // do stuff with rparams.Package
func ParseRequestData(target interface{}, request *Request, reply *Reply) bool {
	err := json.Unmarshal(request.Data, target)
	if err != nil {
		reply.Statuscode = InvalidData
		reply.Statusmsg = fmt.Sprintf("Could not parse request data for %s#%s: %s", request.Agent, request.Action, err)

		return false
	}

	return true
}
