package server

import (
	"context"
	"sync"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
)

func (srv *Instance) handleRawMessage(ctx context.Context, wg *sync.WaitGroup, replies chan *agents.AgentReply, rawmsg *choria.ConnectorMessage) {
	var msg *choria.Message

	transport, err := srv.fw.NewTransportFromJSON(string(rawmsg.Data))
	if err != nil {
		srv.log.Errorf("Could not deceode message into transport: %s", err.Error())
		return
	}

	sreq, err := srv.fw.NewSecureRequestFromTransport(transport, false)
	if err != nil {
		srv.log.Errorf("Could not decode incoming request: %s", err.Error())
		return
	}

	req, err := srv.fw.NewRequestFromSecureRequest(sreq)
	if err != nil {
		srv.log.Errorf("Could not decode secure request: %s", err.Error())
		return
	}

	protocol.CopyFederationData(transport, req)

	if !srv.discovery.ShouldProcess(req, srv.agents.KnownAgents()) {
		srv.log.Debugf("Skipping message %s that does not match local properties", req.RequestID())
		return
	}

	msg, err = choria.NewMessageFromRequest(req, transport.ReplyTo(), srv.fw)
	if err != nil {
		srv.log.Errorf("Could not create Message: %s", err.Error())
		return
	}

	wg.Add(1)
	go srv.agents.Dispatch(ctx, wg, replies, msg, req)

}

func (srv *Instance) handleReply(reply *agents.AgentReply) {
	if reply.Error != nil {
		srv.log.Errorf("Request %s failed, discarding: %s", reply.Message.RequestID, reply.Error.Error())
		return
	}

	msg, err := choria.NewMessageFromRequest(reply.Request, reply.Message.ReplyTo(), srv.fw)
	if err != nil {
		srv.log.Errorf("Cannot create reply Message for %s: %s", reply.Message.RequestID, err.Error())
		return
	}

	msg.Payload = string(reply.Body)

	err = srv.connector.Publish(msg)
	if err != nil {
		srv.log.Errorf("Publishing reply Message for %s failed: %s", reply.Message.RequestID, err.Error())
	}

}

func (srv *Instance) processRequests(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	replies := make(chan *agents.AgentReply, 100)

	for {
		select {
		case rawmsg := <-srv.requests:
			srv.handleRawMessage(ctx, wg, replies, rawmsg)
		case reply := <-replies:
			go srv.handleReply(reply)
		case <-ctx.Done():
			srv.log.Infof("Request processor existing on interrupt")
			return
		}
	}
}
