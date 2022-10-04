// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

type Reply struct {
	Protocol    protocol.ProtocolVersion `json:"protocol"`
	MessageBody string                   `json:"message"`
	Envelope    *ReplyEnvelope           `json:"envelope"`

	mu sync.Mutex
}

type ReplyEnvelope struct {
	RequestID string `json:"requestid"`
	SenderID  string `json:"senderid"`
	Agent     string `json:"agent"`
	Time      int64  `json:"time"`

	seenBy     [][3]string
	federation *FederationTransportHeader
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *Reply) RecordNetworkHop(in string, processor string, out string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.seenBy = append(r.Envelope.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *Reply) NetworkHops() [][3]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.seenBy
}

// SetMessage sets the data to be stored in the Reply.  It should be JSON encoded already.
func (r *Reply) SetMessage(message []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = string(message)
}

// Message retrieves the JSON encoded message set using SetMessage
func (r *Reply) Message() (msg []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return []byte(r.MessageBody)
}

// RequestID retrieves the unique request id
func (r *Reply) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.RequestID
}

// SenderID retrieves the identity of the sending node
func (r *Reply) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.SenderID
}

// Agent retrieves the agent name that sent this reply
func (r *Reply) Agent() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.Agent
}

// Time retrieves the time stamp that this message was made
func (r *Reply) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(r.Envelope.Time, 0)
}

// JSON creates a JSON encoded reply
func (r *Reply) JSON() ([]byte, error) {
	r.mu.Lock()
	j, err := json.Marshal(r)
	r.mu.Unlock()
	if err != nil {
		protocolErrorCtr.Inc()
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	err = r.IsValidJSON(j)
	if err != nil {
		return nil, fmt.Errorf("serialized JSON produced from the Reply does not pass validation: %s", err)
	}

	return j, nil
}

// Version retrieves the protocol version for this message
func (r *Reply) Version() protocol.ProtocolVersion {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *Reply) IsValidJSON(data []byte) (err error) {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(protocol.ReplyV1, data)
	if err != nil {
		return fmt.Errorf("could not validate Reply JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid Reply message: %s", strings.Join(errors, ", "))
	}

	return nil
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *Reply) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		return nil, false
	}

	return r.Envelope.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *Reply) FederationReplyTo() (replyto string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	replyto = r.Envelope.federation.ReplyTo

	return
}

// FederationRequestID retrieves the federation specific requestid
func (r *Reply) FederationRequestID() (id string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		federated = false
		return
	}

	federated = true
	id = r.Envelope.federation.RequestID

	return
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (r *Reply) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *Reply) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *Reply) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Envelope.federation == nil {
		r.Envelope.federation = &FederationTransportHeader{}
	}

	r.Envelope.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *Reply) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Envelope.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *Reply) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Envelope.federation = nil
}
