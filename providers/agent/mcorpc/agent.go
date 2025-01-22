// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/choria-io/go-choria/inter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/audit"
	"github.com/choria-io/go-choria/server/agents"
)

// Action is a function that implements a RPC Action
type Action func(context.Context, *Request, *Reply, *Agent, inter.ConnectorInfo)

// ActivationChecker is a function that can determine if an agent should be activated
type ActivationChecker func() bool

// Agent is an instance of the MCollective compatible RPC agents
type Agent struct {
	Log              *logrus.Entry
	Config           *config.Config
	Choria           inter.Framework
	ServerInfoSource agents.ServerInfoSource

	activationCheck ActivationChecker
	meta            *agents.Metadata
	actions         map[string]Action
}

// New creates a new MCollective SimpleRPC compatible agent
func New(name string, metadata *agents.Metadata, fw inter.Framework, log *logrus.Entry) *Agent {
	a := &Agent{
		meta:            metadata,
		Log:             log.WithFields(logrus.Fields{"agent": name}),
		actions:         make(map[string]Action),
		Choria:          fw,
		Config:          fw.Configuration(),
		activationCheck: func() bool { return true },
	}

	return a
}

// ShouldActivate checks if the agent should be active using the method set in SetActivationChecker
func (a *Agent) ShouldActivate() bool {
	return a.activationCheck()
}

// SetActivationChecker sets the function that can determine if the agent should be active
func (a *Agent) SetActivationChecker(ac ActivationChecker) {
	a.activationCheck = ac
}

// SetServerInfo stores the server info source that owns this agent
func (a *Agent) SetServerInfo(si agents.ServerInfoSource) {
	a.ServerInfoSource = si
}

// ServerInfo returns the stored server info source
func (a *Agent) ServerInfo() agents.ServerInfoSource {
	return a.ServerInfoSource
}

// RegisterAction registers an action into the agent
func (a *Agent) RegisterAction(name string, f Action) error {
	if _, ok := a.actions[name]; ok {
		return fmt.Errorf("cannot register action %s, it already exist", name)
	}

	a.actions[name] = f

	return nil
}

// MustRegisterAction registers an action and panics if it fails
func (a *Agent) MustRegisterAction(name string, f Action) {
	if _, ok := a.actions[name]; ok {
		panic(fmt.Errorf("cannot register action %s, it already exist", name))
	}

	a.actions[name] = f
}

// HandleMessage attempts to parse a choria.Message as a MCollective SimpleRPC request and calls
// the agents and actions associated with it
func (a *Agent) HandleMessage(ctx context.Context, msg inter.Message, request protocol.Request, conn inter.ConnectorInfo, outbox chan *agents.AgentReply) {
	var err error

	reply := a.newReply()
	defer a.publish(reply, msg, request, outbox)

	rpcrequest, err := a.parseIncomingMessage(msg.Payload(), request)
	if err != nil {
		reply.Statuscode = InvalidData
		reply.Statusmsg = fmt.Sprintf("Could not process request: %s", err)
		return
	}

	reply.Action = rpcrequest.Action

	action, found := a.actions[rpcrequest.Action]
	if !found {
		reply.Statuscode = UnknownAction
		reply.Statusmsg = fmt.Sprintf("Unknown action %s for agent %s", rpcrequest.Action, a.Name())
		return
	}

	if a.Config.RPCAuthorization {
		if !a.authorize(rpcrequest) {
			a.Log.Warnf("Denying %s access to %s#%s based on authorization policy for request %s", request.CallerID(), rpcrequest.Agent, rpcrequest.Action, request.RequestID())
			reply.Statuscode = Aborted
			reply.Statusmsg = "You are not authorized to call this agent or action"
			return
		}
	}

	if a.Config.RPCAudit {
		audit.Request(request, rpcrequest.Agent, rpcrequest.Action, rpcrequest.Data, a.Config)
	}

	a.Log.Infof("Handling message %s for %s#%s from %s", msg.RequestID(), a.Name(), rpcrequest.Action, request.CallerID())

	action(ctx, rpcrequest, reply, a, conn)
}

// Name retrieves the name of the agent
func (a *Agent) Name() string {
	return a.meta.Name
}

// ActionNames returns a list of known actions in the agent
func (a *Agent) ActionNames() []string {
	var actions []string

	for k := range a.actions {
		actions = append(actions, k)
	}

	sort.Strings(actions)

	return actions
}

// Metadata retrieves the agent metadata
func (a *Agent) Metadata() *agents.Metadata {
	return a.meta
}

func (a *Agent) publish(rpcreply *Reply, msg inter.Message, request protocol.Request, outbox chan *agents.AgentReply) {
	if rpcreply.DisableResponse {
		return
	}

	reply := &agents.AgentReply{
		Message: msg,
		Request: request,
	}

	if rpcreply.Data == nil {
		rpcreply.Data = "{}"
	}

	j, err := json.Marshal(rpcreply)
	if err != nil {
		a.Log.Errorf("Could not JSON encode reply: %s", err)
		reply.Error = err
	}

	reply.Body = j

	outbox <- reply
}

func (a *Agent) newReply() *Reply {
	reply := &Reply{
		Statuscode: OK,
		Statusmsg:  "OK",
		Data:       json.RawMessage(`{}`),
	}

	return reply
}

func (a *Agent) parseIncomingMessage(msg []byte, request protocol.Request) (*Request, error) {
	r := &Request{}

	err := json.Unmarshal(msg, r)
	if err != nil {
		return nil, fmt.Errorf("could not parse incoming message as a MCollective SimpleRPC Request: %s", err)
	}

	r.CallerID = request.CallerID()
	r.RequestID = request.RequestID()
	r.SenderID = request.SenderID()
	r.Collective = request.Collective()
	r.CallerPublicData = request.CallerPublicData()
	r.SignerPublicData = request.SignerPublicData()
	r.TTL = request.TTL()
	r.Time = request.Time()
	r.Filter, _ = request.Filter()

	if r.Data == nil {
		r.Data = json.RawMessage(`{}`)
	}

	return r, nil
}

func (a *Agent) authorize(req *Request) bool {
	if req.Agent != a.Name() {
		a.Log.Errorf("Could not process authorization for request for a different agent")
		return false
	}

	return AuthorizeRequest(a.Choria, req, a.Config, a.ServerInfoSource, a.Log)
}

// AuthorizeRequest authorizes a request using the configured authorizer
func AuthorizeRequest(fw inter.Framework, req *Request, cfg *config.Config, si agents.ServerInfoSource, log *logrus.Entry) bool {
	if cfg == nil {
		log.Errorf("Could not process authorization without a configuration")
		return false
	}
	if !cfg.RPCAuthorization {
		return true
	}
	if req == nil {
		log.Errorf("Could not process authorization without a request")
		return false
	}
	if req.Agent == "" {
		log.Errorf("Could not process authorization without a agent name")
		return false
	}
	if si == nil {
		log.Errorf("Could not process authorization without a server info source")
		return false
	}

	prov := strings.ToLower(cfg.RPCAuthorizationProvider)

	switch prov {
	case "action_policy":
		return actionPolicyAuthorize(req, cfg, log)

	case "rego_policy":
		auth, err := regoPolicyAuthorize(req, fw, si, cfg, log)
		if err != nil {
			log.Errorf("Could not process Open Policy Agent policy: %v", err)
			return false
		}
		return auth

	case "aaasvc", "aaasvc_policy":
		auth, err := aaasvcPolicyAuthorize(req, cfg, log)
		if err != nil {
			log.Errorf("Could not process JWT policy: %v", err)
			return false
		}
		return auth

	default:
		log.Errorf("Unsupported authorization provider: %s", prov)

	}

	return false
}
