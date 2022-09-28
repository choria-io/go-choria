// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"

	"github.com/choria-io/go-choria/server/agents"
)

func (srv *Instance) handleRawMessage(ctx context.Context, wg *sync.WaitGroup, replies chan *agents.AgentReply, rawmsg inter.ConnectorMessage) {
	var msg inter.Message

	totalCtr.WithLabelValues(srv.cfg.Identity).Inc()

	transport, err := srv.fw.NewTransportFromJSON(rawmsg.Data())
	if err != nil {
		srv.log.Errorf("Could not deceode message into transport: %s", err)
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		return
	}

	sreq, err := srv.fw.NewSecureRequestFromTransport(transport, srv.cfg.DisableSecurityProviderVerify)
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

	if !srv.discovery.ShouldProcess(req) {
		filteredCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Debugf("Skipping message %s that does not match local properties", req.RequestID())
		return
	}

	passedCtr.WithLabelValues(srv.cfg.Identity).Inc()

	msg, err = srv.fw.NewMessageFromRequest(req, transport.ReplyTo())
	if err != nil {
		unvalidatedCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Could not create Message: %s", err)
		return
	}

	if !msg.ValidateTTL() {
		ttlExpiredCtr.WithLabelValues(srv.cfg.Identity).Inc()
		srv.log.Errorf("Message %s created at %s is too old, TTL is %d", msg.String(), msg.TimeStamp(), msg.TTL())
		return
	}

	validatedCtr.WithLabelValues(srv.cfg.Identity).Inc()

	srv.lastMsgProcessed = time.Now()

	wg.Add(1)
	go srv.agents.Dispatch(ctx, wg, replies, msg, req)
}

func (srv *Instance) handleReply(reply *agents.AgentReply) {
	if reply.Error != nil {
		srv.log.Errorf("Request %s failed, discarding: %s", reply.Message.RequestID(), reply.Error.Error())
		return
	}

	msg, err := srv.fw.NewMessageFromRequest(reply.Request, reply.Message.ReplyTo())
	if err != nil {
		srv.log.Errorf("Cannot create reply Message for %s: %s", reply.Message.RequestID(), err)
		return
	}

	msg.SetPayload(reply.Body)

	err = srv.connector.Publish(msg)
	if err != nil {
		srv.log.Errorf("Publishing reply Message for %s failed: %s", reply.Message.RequestID(), err)
	}

	repliesCtr.WithLabelValues(srv.cfg.Identity).Inc()
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
