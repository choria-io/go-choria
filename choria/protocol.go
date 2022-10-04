// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"fmt"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/protocol"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	v2 "github.com/choria-io/go-choria/protocol/v2"
)

// NewMessage creates a new Message associated with this Choria instance
func (fw *Framework) NewMessage(payload []byte, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error) {
	return message.NewMessage(payload, agent, collective, msgType, request, fw)
}

// RequestProtocol determines the protocol version to use based on security provider technology
func (fw *Framework) RequestProtocol() protocol.ProtocolVersion {
	switch fw.security.BackingTechnology() {
	case inter.SecurityTechnologyX509:
		return protocol.RequestV1
	case inter.SecurityTechnologyED25519JWT:
		return protocol.RequestV2
	}

	return protocol.Unknown
}

// NewRequestMessageFromTransportJSON creates a Message from a Transport JSON that holds a Request
func (fw *Framework) NewRequestMessageFromTransportJSON(payload []byte) (inter.Message, error) {
	transport, err := fw.NewTransportFromJSON(payload)
	if err != nil {
		return nil, err
	}

	srequest, err := fw.NewSecureRequestFromTransport(transport, false)
	if err != nil {
		return nil, err
	}

	request, err := fw.NewRequestFromSecureRequest(srequest)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, request)

	msg, err := message.NewMessageFromRequest(request, transport.ReplyTo(), fw)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (fw *Framework) NewMessageFromRequest(req protocol.Request, replyto string) (inter.Message, error) {
	return message.NewMessageFromRequest(req, replyto, fw)
}

// NewReplyFromTransportJSON creates a new Reply from a transport JSON
func (fw *Framework) NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error) {
	transport, err := fw.NewTransportFromJSON(payload)
	if err != nil {
		return nil, err
	}

	sreply, err := fw.NewSecureReplyFromTransport(transport, skipvalidate)
	if err != nil {
		return nil, err
	}

	reply, err := fw.NewReplyFromSecureReply(sreply)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, reply)

	return reply, nil
}

// NewRequestFromTransportJSON creates a new Request from transport JSON
func (fw *Framework) NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error) {
	transport, err := fw.NewTransportFromJSON(payload)
	if err != nil {
		return nil, err
	}

	sreq, err := fw.NewSecureRequestFromTransport(transport, skipvalidate)
	if err != nil {
		return nil, err
	}

	req, err := fw.NewRequestFromSecureRequest(sreq)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, req)

	return req, nil
}

// NewRequest creates a new Request complying with a specific protocol version like protocol.RequestV1
func (fw *Framework) NewRequest(version protocol.ProtocolVersion, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
	switch version {
	case protocol.RequestV1:
		request, err = v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	case protocol.RequestV2:
		request, err = v2.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	default:
		err = fmt.Errorf("do not know how to create a Request version %s", version)
	}

	return request, err
}

// NewRequestFromMessage creates a new Request with the Message settings preloaded complying with a specific protocol version like protocol.RequestV1
func (fw *Framework) NewRequestFromMessage(version protocol.ProtocolVersion, msg inter.Message) (req protocol.Request, err error) {
	if !(msg.Type() == inter.RequestMessageType || msg.Type() == inter.DirectRequestMessageType || msg.Type() == inter.ServiceRequestMessageType) {
		err = fmt.Errorf("cannot use '%s' message to construct a Request", msg.Type())
		return nil, err
	}

	req, err = fw.NewRequest(version, msg.Agent(), msg.SenderID(), msg.CallerID(), msg.TTL(), msg.RequestID(), msg.Collective())
	if err != nil {
		return nil, fmt.Errorf("could not create a Request from a Message: %s", err)
	}

	req.SetMessage(msg.Payload())

	if msg.Filter() == nil {
		req.NewFilter()
	} else {
		req.SetFilter(msg.Filter())
	}

	return req, nil
}

// NewReply creates a new Reply, the version will match that of the given request
func (fw *Framework) NewReply(request protocol.Request) (reply protocol.Reply, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		return v1.NewReply(request, fw.Config.Identity)
	case protocol.RequestV2:
		return v2.NewReply(request, fw.Config.Identity)
	default:
		return nil, fmt.Errorf("do not know how to create a Reply version %s", request.Version())
	}
}

// NewReplyFromMessage creates a new Reply with the Message settings preloaded complying with a specific protocol version like protocol.ReplyV1
func (fw *Framework) NewReplyFromMessage(version protocol.ProtocolVersion, msg inter.Message) (rep protocol.Reply, err error) {
	if msg.Type() != "reply" {
		return nil, fmt.Errorf("cannot use '%s' message to construct a Reply", msg.Type())
	}

	if msg.Request() == nil {
		return nil, fmt.Errorf("cannot create a Reply from Messages without Requests")
	}

	req, err := fw.NewRequestFromMessage(version, msg.Request())
	if err != nil {
		return nil, err
	}

	rep, err = fw.NewReply(req)
	rep.SetMessage(msg.Payload())

	return rep, err
}

// NewReplyFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (fw *Framework) NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error) {
	switch sr.Version() {
	case protocol.SecureReplyV1:
		return v1.NewReplyFromSecureReply(sr)
	case protocol.SecureReplyV2:
		return v2.NewReplyFromSecureReply(sr)
	default:
		return nil, fmt.Errorf("do not know how to create a Reply version %s", sr.Version())
	}
}

// NewRequestFromSecureRequest creates a new Request from a SecureRequest, the version will match what is in the JSON payload
func (fw *Framework) NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error) {
	switch sr.Version() {
	case protocol.SecureRequestV1:
		return v1.NewRequestFromSecureRequest(sr)
	case protocol.SecureRequestV2:
		return v2.NewRequestFromSecureRequest(sr)
	default:
		return nil, fmt.Errorf("do not know how to create a Reply version %s", sr.Version())
	}

}

// NewSecureReply creates a new SecureReply with the given Reply message as payload
func (fw *Framework) NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error) {
	switch reply.Version() {
	case protocol.ReplyV1:
		return v1.NewSecureReply(reply, fw.security)
	case protocol.ReplyV2:
		return v2.NewSecureReply(reply, fw.security)
	default:
		return nil, fmt.Errorf("do not know how to create a SecureReply based on a Reply version %s", reply.Version())
	}

}

// NewSecureReplyFromTransport creates a new SecureReply from the JSON payload of TransportMessage, the version SecureReply will be the same as the TransportMessage
func (fw *Framework) NewSecureReplyFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureReply, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		return v1.NewSecureReplyFromTransport(message, fw.security, skipvalidate)
	case protocol.TransportV2:
		return v2.NewSecureReplyFromTransport(message, fw.security, skipvalidate)
	default:
		return nil, fmt.Errorf("do not know how to create a SecureReply version %s", message.Version())
	}
}

// NewSecureRequest creates a new SecureRequest with the given Request message as payload
func (fw *Framework) NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		if fw.security.IsRemoteSigning() {
			return v1.NewRemoteSignedSecureRequest(request, fw.security)
		}

		return v1.NewSecureRequest(request, fw.security)
	case protocol.RequestV2:
		if fw.security.IsRemoteSigning() {
			return v2.NewRemoteSignedSecureRequest(request, fw.security)
		}

		return v2.NewSecureRequest(request, fw.security)
	default:
		return nil, fmt.Errorf("do not know how to create a SecureReply from a Request with version %s", request.Version())
	}
}

// NewSecureRequestFromTransport creates a new SecureRequest from the JSON payload of TransportMessage, the version SecureRequest will be the same as the TransportMessage
func (fw *Framework) NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		return v1.NewSecureRequestFromTransport(message, fw.security, skipvalidate)
	case protocol.TransportV2:
		return v2.NewSecureRequestFromTransport(message, fw.security, skipvalidate)
	default:
		return nil, fmt.Errorf("do not know how to create a SecureReply from a TransportMessage version %s", message.Version())
	}
}

// NewTransportForSecureRequest creates a new TransportMessage with a SecureRequest as payload.  The Transport will be the same version as the SecureRequest
func (fw *Framework) NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error) {
	switch request.Version() {
	case protocol.SecureRequestV1:
		message, err = v1.NewTransportMessage(fw.Config.Identity)
	case protocol.SecureRequestV2:
		message, err = v2.NewTransportMessage(fw.Config.Identity)
	default:
		return nil, fmt.Errorf("co not know how to create a Transport message for SecureRequest version %s", request.Version())
	}

	if err != nil {
		fw.log.Errorf("Failed to create transport from secure request: %s", err)
		return nil, err
	}

	err = message.SetRequestData(request)
	if err != nil {
		fw.log.Errorf("Failed to create transport from secure request: %s", err)
		return nil, err
	}

	return message, nil
}

// NewTransportForSecureReply creates a new TransportMessage with a SecureReply as payload.  The Transport will be the same version as the SecureRequest
func (fw *Framework) NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error) {
	switch reply.Version() {
	case protocol.SecureReplyV1:
		message, err = v1.NewTransportMessage(fw.Config.Identity)
	case protocol.SecureReplyV2:
		message, err = v2.NewTransportMessage(fw.Config.Identity)
	default:
		return nil, fmt.Errorf("do not know how to create a Transport message for SecureRequest version %s", reply.Version())
	}

	if err != nil {
		return nil, err
	}

	message.SetReplyData(reply)

	return message, nil
}

// NewReplyTransportForMessage creates a new Transport message based on a Message and the request its a reply to
//
// The new transport message will have the same version as the request its based on
func (fw *Framework) NewReplyTransportForMessage(msg inter.Message, request protocol.Request) (protocol.TransportMessage, error) {
	reply, err := fw.NewReply(request)
	if err != nil {
		return nil, fmt.Errorf("could not create Reply: %s", err)
	}

	reply.SetMessage(msg.Payload())

	sreply, err := fw.NewSecureReply(reply)
	if err != nil {
		return nil, fmt.Errorf("could not create Secure Reply: %s", err)
	}

	transport, err := fw.NewTransportForSecureReply(sreply)
	if err != nil {
		return nil, fmt.Errorf("could not create Transport: %s", err)
	}

	protocol.CopyFederationData(request, transport)

	return transport, nil
}

// NewRequestTransportForMessage creates a new versioned Transport message based on a Message
func (fw *Framework) NewRequestTransportForMessage(msg inter.Message, version protocol.ProtocolVersion) (protocol.TransportMessage, error) {
	req, err := fw.NewRequestFromMessage(version, msg)
	if err != nil {
		return nil, fmt.Errorf("could not create Request: %s", err)
	}

	sr, err := fw.NewSecureRequest(req)
	if err != nil {
		return nil, err
	}

	transport, err := fw.NewTransportForSecureRequest(sr)
	if err != nil {
		return nil, fmt.Errorf("could not create Transport: %s", err)
	}

	return transport, nil
}

// NewTransportMessage creates a new TransportMessage complying with a specific protocol version like protocol.TransportV1
func (fw *Framework) NewTransportMessage(version protocol.ProtocolVersion) (message protocol.TransportMessage, err error) {
	switch version {
	case protocol.TransportV1:
		return v1.NewTransportMessage(fw.Config.Identity)
	case protocol.TransportV2:
		return v2.NewTransportMessage(fw.Config.Identity)
	default:
		return nil, fmt.Errorf("so not know how to create a Transport version '%s'", version)
	}
}

// NewTransportFromJSON creates a new TransportMessage from a JSON payload.  The version will match what is in the payload
func (fw *Framework) NewTransportFromJSON(data []byte) (message protocol.TransportMessage, err error) {
	switch protocol.VersionFromJSON(data) {
	case protocol.TransportV1:
		return v1.NewTransportFromJSON(data)
	case protocol.TransportV2:
		return v2.NewTransportFromJSON(data)
	default:
		return nil, fmt.Errorf("do not know how to create a TransportMessage from an expected JSON format message with content: %s", data)
	}
}
