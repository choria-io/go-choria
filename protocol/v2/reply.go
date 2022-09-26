// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/protocol"
)

// NewReply creates a io.choria.protocol.v2.request based on a previous Request
func NewReply(request protocol.Request, certName string) (protocol.Reply, error) {
	if request.Version() != protocol.RequestV2 {
		return nil, fmt.Errorf("cannot create a version 2 Reply from a %s request", request.Version())
	}

	rep := &reply{
		Protocol: protocol.ReplyV2,
		replyEnvelope: replyEnvelope{
			Request: request.RequestID(),
			Sender:  certName,
			Agent:   request.Agent(),
			Time:    time.Now().Unix(),
		},
	}

	protocol.CopyFederationData(request, rep)

	j, err := request.JSON()
	if err != nil {
		return nil, fmt.Errorf("could not turn Request %s into a JSON document: %s", request.RequestID(), err)
	}

	rep.SetMessage(j)

	return rep, nil
}

type reply struct {
	Protocol    string `json:"protocol"`
	MessageBody []byte `json:"message"`

	replyEnvelope

	mu sync.Mutex
}

type replyEnvelope struct {
	Request string `json:"request"`
	Sender  string `json:"sender"`
	Agent   string `json:"agent"`
	Time    int64  `json:"time"`

	seenBy     [][3]string
	federation *federationTransportHeader
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (r *reply) RecordNetworkHop(in string, processor string, out string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seenBy = append(r.seenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (r *reply) NetworkHops() [][3]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.seenBy
}

// SetMessage sets the data to be stored in the Reply
func (r *reply) SetMessage(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.MessageBody = []byte(message)
}

// Message retrieves the JSON encoded message set using SetMessage
func (r *reply) Message() (msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return string(r.MessageBody)
}

// RequestID retrieves the unique request id
func (r *reply) RequestID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Request
}

// SenderID retrieves the identity of the sending node
func (r *reply) SenderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Sender
}

// Agent retrieves the agent name that sent this reply
func (r *reply) Agent() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.replyEnvelope.Agent
}

// Time retrieves the time stamp that this message was made
func (r *reply) Time() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return time.Unix(r.replyEnvelope.Time, 0)
}

// JSON creates a JSON encoded reply
func (r *reply) JSON() (body string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		return "", fmt.Errorf("could not JSON Marshal: %s", err)
	}

	body = string(j)

	err = r.isValidJSONUnlocked(body)
	if err != nil {
		return "", fmt.Errorf("serialized JSON produced from the Reply does not pass validation: %s", err)
	}

	return body, nil
}

// Version retrieves the protocol version for this message
func (r *reply) Version() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.Protocol
}

func (r *reply) isValidJSONUnlocked(data string) error {
	if !protocol.ClientStrictValidation {
		return nil
	}

	// TODO
	// _, errors, err := schemaValidate(replySchema, data)
	// if err != nil {
	// 	return fmt.Errorf("could not validate Reply JSON data: %s", err)
	// }
	//
	// if len(errors) != 0 {
	// 	return fmt.Errorf("supplied JSON document is not a valid Reply message: %s", strings.Join(errors, ", "))
	// }

	return nil
}

// IsValidJSON validates the given JSON data against the schema
func (r *reply) IsValidJSON(data string) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isValidJSONUnlocked(data)
}

// FederationTargets retrieves the list of targets this message is destined for
func (r *reply) FederationTargets() (targets []string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.replyEnvelope.federation == nil {
		return nil, false
	}

	return r.federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (r *reply) FederationReplyTo() (replyto string, federated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		return "", false
	}

	return r.federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (r *reply) FederationRequestID() (id string, federated bool) {
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
func (r *reply) SetFederationTargets(targets []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &federationTransportHeader{}
	}

	r.federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (r *reply) SetFederationReplyTo(reply string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &federationTransportHeader{}
	}

	r.federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (r *reply) SetFederationRequestID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.federation == nil {
		r.federation = &federationTransportHeader{}
	}

	r.federation.RequestID = id
}

// IsFederated determines if this message is federated
func (r *reply) IsFederated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.federation != nil
}

// SetUnfederated removes any federation information from the message
func (r *reply) SetUnfederated() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.federation = nil
}
