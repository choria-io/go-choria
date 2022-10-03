// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

type Reply struct {
	// The protocol version for this transport `io.choria.protocol.v2.reply` / protocol.ReplyV2
	Protocol string `json:"protocol"`
	// The arbitrary data contained in the reply - like a RPC reply
	MessageBody []byte `json:"message"`
	// The ID of the request this reply relates to
	Request string `json:"request"`
	// The host sending the reply
	Sender string `json:"sender"`
	// The agent the reply originates from
	SendingAgent string `json:"agent"`
	// The unix nano time the request was created
	TimeStamp int64 `json:"time"`

	seenBy     [][3]string
	federation *FederationTransportHeader

	mu sync.Mutex
}

// NewReply creates a io.choria.protocol.v2.request based on a previous Request
func NewReply(request protocol.Request, certName string) (protocol.Reply, error) {
	if request.Version() != protocol.RequestV2 {
		return nil, fmt.Errorf("cannot create a version 2 Reply from a %s request", request.Version())
	}

	rep := &Reply{
		Protocol:     protocol.ReplyV2,
		Request:      request.RequestID(),
		Sender:       certName,
		SendingAgent: request.Agent(),
		TimeStamp:    time.Now().UnixNano(),
	}

	protocol.CopyFederationData(request, rep)

	j, err := request.JSON()
	if err != nil {
		return nil, fmt.Errorf("could not turn Request %s into a JSON document: %s", request.RequestID(), err)
	}

	rep.SetMessage(j)

	return rep, nil
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *Reply) RecordNetworkHop(in string, processor string, out string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seenBy = append(r.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *Reply) NetworkHops() [][3]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.seenBy
}

// SetMessage sets the data to be stored in the Reply
func (r *Reply) SetMessage(message []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = message
}

// Message retrieves the JSON encoded message set using SetMessage
func (r *Reply) Message() (msg []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.MessageBody
}

// RequestID retrieves the unique request id
func (r *Reply) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Request
}

// SenderID retrieves the identity of the sending node
func (r *Reply) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Sender
}

// Agent retrieves the agent name that sent this reply
func (r *Reply) Agent() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.SendingAgent
}

// Time retrieves the time stamp that this message was made
func (r *Reply) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(0, r.TimeStamp)
}

// JSON creates a JSON encoded reply
func (r *Reply) JSON() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	err = r.isValidJSONUnlocked(j)
	if err != nil {
		return nil, fmt.Errorf("serialized JSON produced from the Reply does not pass validation: %s", err)
	}

	return j, nil
}

// Version retrieves the protocol version for this message
func (r *Reply) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

func (r *Reply) isValidJSONUnlocked(data []byte) error {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(protocol.ReplyV2, data)
	if err != nil {
		return fmt.Errorf("could not validate Reply JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, strings.Join(errors, ", "))
	}

	return nil
}

// IsValidJSON validates the given JSON data against the schema
func (r *Reply) IsValidJSON(data []byte) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isValidJSONUnlocked(data)
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *Reply) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return nil, false
	}

	return r.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *Reply) FederationReplyTo() (replyto string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return "", false
	}

	return r.federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (r *Reply) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return "", false
	}

	return r.federation.RequestID, true
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *Reply) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &FederationTransportHeader{}
	}

	r.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *Reply) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &FederationTransportHeader{}
	}

	r.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *Reply) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &FederationTransportHeader{}
	}

	r.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *Reply) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *Reply) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.federation = nil
}
