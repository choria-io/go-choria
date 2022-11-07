// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/protocol"
)

type TransportMessage struct {
	// The protocol version for this transport `io.choria.protocol.v2.transport` / protocol.TransportV2
	Protocol protocol.ProtocolVersion `json:"protocol"`
	// The payload to be transport, a Secure Request or Secure Reply
	Data []byte `json:"data"`
	// Optional headers
	Headers *TransportHeaders `json:"headers,omitempty"`

	mu sync.Mutex
}

type TransportHeaders struct {
	// A transport specific response channel for this message, used in requests
	ReplyTo string `json:"reply,omitempty"`
	// The host that sent this message
	Sender string `json:"sender,omitempty"`
	// A trace of host/broker pairs that the message traversed
	SeenBy [][3]string `json:"trace,omitempty"`
	// Headers to assist federation
	Federation *FederationTransportHeader `json:"federation,omitempty"`
}

type FederationTransportHeader struct {
	// The request ID a federated message belongs to
	RequestID string `json:"request,omitempty"`
	// The original `reply` before federation
	ReplyTo string `json:"reply,omitempty"`
	// The identities who the federated message is for
	Targets []string `json:"targets,omitempty"`
}

// NewTransportMessage creates a io.choria.protocol.v2.transport
func NewTransportMessage(sender string) (message protocol.TransportMessage, err error) {
	message = &TransportMessage{
		Protocol: protocol.TransportV2,
		Headers:  &TransportHeaders{},
	}

	message.SetSender(sender)

	return message, nil
}

// NewTransportFromJSON creates a new TransportMessage from JSON
func NewTransportFromJSON(data []byte) (message protocol.TransportMessage, err error) {
	message = &TransportMessage{
		Headers: &TransportHeaders{},
	}

	err = message.IsValidJSON(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (m *TransportMessage) SetFederationRequestID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.RequestID = id
}

func (m *TransportMessage) SetFederationReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.ReplyTo = reply
}

func (m *TransportMessage) SetFederationTargets(targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		m.Headers.Federation = &FederationTransportHeader{}
	}

	m.Headers.Federation.Targets = targets
}

func (m *TransportMessage) SetUnfederated() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.Federation = nil
}

func (m *TransportMessage) FederationRequestID() (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return "", false
	}

	return m.Headers.Federation.RequestID, true
}

func (m *TransportMessage) FederationReplyTo() (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return "", false
	}

	return m.Headers.Federation.ReplyTo, true
}

func (m *TransportMessage) FederationTargets() ([]string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Headers.Federation == nil {
		return nil, false
	}

	return m.Headers.Federation.Targets, true
}

func (m *TransportMessage) RecordNetworkHop(in string, processor string, out string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.SeenBy = append(m.Headers.SeenBy, [3]string{in, processor, out})
}

func (m *TransportMessage) NetworkHops() [][3]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.SeenBy
}

func (m *TransportMessage) IsFederated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.Federation != nil
}

func (m *TransportMessage) SetReplyData(reply protocol.SecureReply) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		return err
	}

	m.Data = j

	return nil
}

func (m *TransportMessage) SetRequestData(request protocol.SecureRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	j, err := request.JSON()
	if err != nil {
		return err
	}

	m.Data = j

	return nil
}

func (m *TransportMessage) SetReplyTo(reply string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.ReplyTo = reply
}

func (m *TransportMessage) SetSender(sender string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Headers.Sender = sender
}

func (m *TransportMessage) ReplyTo() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.ReplyTo
}

func (m *TransportMessage) SenderID() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.Sender
}

func (m *TransportMessage) SeenBy() [][3]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Headers.SeenBy
}

func (m *TransportMessage) Message() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Data, nil
}

func (m *TransportMessage) IsValidJSON(data []byte) error {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemaValidate(protocol.TransportV2, data)
	if err != nil {
		return err
	}

	if len(errors) != 0 {
		return fmt.Errorf("%w: %s", ErrInvalidJSON, strings.Join(errors, ", "))
	}

	return nil
}

func (m *TransportMessage) JSON() ([]byte, error) {
	m.mu.Lock()
	j, err := json.Marshal(m)
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}

	if err = m.IsValidJSON(j); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidJSON, err)
	}

	return j, nil
}

func (m *TransportMessage) Version() protocol.ProtocolVersion {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Protocol
}
