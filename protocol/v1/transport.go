// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/protocol"
)

type TransportMessage struct {
	Protocol protocol.ProtocolVersion `json:"protocol"`
	Data     []byte                   `json:"data"`
	Headers  *TransportHeaders        `json:"headers"`

	mu sync.Mutex
}

type TransportHeaders struct {
	ReplyTo           string                     `json:"reply-to,omitempty"`
	MCollectiveSender string                     `json:"mc_sender,omitempty"`
	SeenBy            [][3]string                `json:"seen-by,omitempty"`
	Federation        *FederationTransportHeader `json:"federation,omitempty"`
}

type FederationTransportHeader struct {
	RequestID string   `json:"req,omitempty"`
	ReplyTo   string   `json:"reply-to,omitempty"`
	Targets   []string `json:"target,omitempty"`
}

// Message retrieves the stored data
func (m *TransportMessage) Message() (data []byte, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Data, nil
}

// IsFederated determines if this message is federated
func (m *TransportMessage) IsFederated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.Federation != nil
}

// FederationTargets retrieves the list of targets this message is destined for
func (m *TransportMessage) FederationTargets() (targets []string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return nil, false
	}

	return m.Headers.Federation.Targets, true
}

// FederationReplyTo retrieves the reply to string set by the federation broker
func (m *TransportMessage) FederationReplyTo() (replyto string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return "", false
	}

	return m.Headers.Federation.ReplyTo, true
}

// FederationRequestID retrieves the federation specific requestid
func (m *TransportMessage) FederationRequestID() (id string, federated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return "", false
	}

	return m.Headers.Federation.RequestID, true
}

// SenderID retrieves the identity of the sending host
func (m *TransportMessage) SenderID() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.MCollectiveSender
}

// ReplyTo retrieves the destination description where replies should go to
func (m *TransportMessage) ReplyTo() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.ReplyTo
}

// SeenBy retrieves the list of end points that this messages passed thruogh
func (m *TransportMessage) SeenBy() [][3]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.SeenBy
}

// SetFederationTargets sets the list of hosts this message should go to.
//
// Federation brokers will duplicate the message and send one for each target
func (m *TransportMessage) SetFederationTargets(targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.Targets = targets
}

// SetFederationReplyTo stores the original reply-to destination in the federation headers
func (m *TransportMessage) SetFederationReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.ReplyTo = reply
}

// SetFederationRequestID sets the request ID for federation purposes
func (m *TransportMessage) SetFederationRequestID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.RequestID = id
}

// SetSender sets the "mc_sender" - typically the identity of the sending host
func (m *TransportMessage) SetSender(sender string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.MCollectiveSender = sender
}

// SetReplyTo sets the reply-to targget
func (m *TransportMessage) SetReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.ReplyTo = reply
}

// SetReplyData extracts the JSON body from a SecureReply and stores it
func (m *TransportMessage) SetReplyData(reply protocol.SecureReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		return fmt.Errorf("could not JSON encode the Reply structure for transport: %s", err)
	}

	m.Data = j

	return nil
}

// SetRequestData extracts the JSON body from a SecureRequest and stores it
func (m *TransportMessage) SetRequestData(request protocol.SecureRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		return fmt.Errorf("could not JSON encode the Request structure for transport: %s", err)
	}

	m.Data = j

	return nil
}

// RecordNetworkHop appends a hop onto the list of those who processed this message
func (m *TransportMessage) RecordNetworkHop(in string, processor string, out string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.SeenBy = append(m.Headers.SeenBy, [3]string{in, processor, out})
}

// NetworkHops returns a list of tuples this messaged traveled through
func (m *TransportMessage) NetworkHops() [][3]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.SeenBy
}

// JSON creates a JSON encoded message
func (m *TransportMessage) JSON() ([]byte, error) {
	m.mu.Lock()
	j, err := json.Marshal(m)
	m.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("could not JSON Marshal: %s", err)
	}

	if err = m.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("the JSON produced from the Transport does not pass validation: %s", err)
	}

	return j, nil
}

// SetUnfederated removes any federation information from the message
func (m *TransportMessage) SetUnfederated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.Federation = nil
}

// Version retrieves the protocol version for this message
func (m *TransportMessage) Version() protocol.ProtocolVersion {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Protocol
}

// IsValidJSON validates the given JSON data against the Transport schema
func (m *TransportMessage) IsValidJSON(data []byte) error {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(protocol.TransportV1, data)
	if err != nil {
		return fmt.Errorf("could not validate Transport JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid Transport message: %s", strings.Join(errors, ", "))
	}

	return nil
}
