package server

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server/agents"
)

func (srv *Instance) handleRawMessage(ctx context.Context, wg *sync.WaitGroup, replies chan *agents.AgentReply, rawmsg *choria.ConnectorMessage) {
	var msg *choria.Message

	totalCtr.WithLabelValues(srv.cfg.Identity).Inc()

	transport, err := srv.fw.NewTransportFromJSON(string(rawmsg.Data))
	if err != nil {
		srv.log.Errorf("Could not deceode message into transport: %s", err)
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		return
	}

	sreq, err := srv.fw.NewSecureRequestFromTransport(transport, false)
	if err != nil {
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Could not decode incoming request: %s", err)
		return
	}

	req, err := srv.fw.NewRequestFromSecureRequest(sreq)
	if err != nil {
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Could not decode secure request: %s", err)
		return
	}

	protocol.CopyFederationData(transport, req)

	if !srv.discovery.ShouldProcess(req, srv.agents.KnownAgents()) {
		filteredCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Debugf("Skipping message %s that does not match local properties", req.RequestID())
		return
	}

	passedCtr.WithLabelValues(srv.cfg.Identity).Inc()

	msg, err = choria.NewMessageFromRequest(req, transport.ReplyTo(), srv.fw)
	if err != nil {
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Could not create Message: %s", err)
		return
	}

	if !msg.ValidateTTL() {
		ttlExpiredCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Message %s created at %s is too old, TTL is %d", msg.String(), msg.TimeStamp, msg.TTL)
		return
	}

	validatedCtr.WithLabelValues(srv.cfg.Identity).Inc()

	srv.lastMsgProcessed = time.Now()

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
		srv.log.Errorf("Cannot create reply Message for %s: %s", reply.Message.RequestID, err)
		return
	}

	msg.Payload = string(reply.Body)

	err = srv.connector.Publish(msg)
	if err != nil {
		srv.log.Errorf("Publishing reply Message for %s failed: %s", reply.Message.RequestID, err)
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

			srv.publichShutdownEvent()
			srv.connector.Close()

			return
		}
	}
}
