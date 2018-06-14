package choria

import (
	"fmt"

	"github.com/choria-io/go-protocol/protocol"
	"github.com/choria-io/go-protocol/protocol/v1"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// NewMessage creates a new Message associated with this Choria instance
func (fw *Framework) NewMessage(payload string, agent string, collective string, msgType string, request *Message) (msg *Message, err error) {
	msg, err = NewMessage(payload, agent, collective, msgType, request, fw)

	return
}

// NewRequestMessageFromTransportJSON creates a Message from a Transport JSON that holds a Request
func (fw *Framework) NewRequestMessageFromTransportJSON(payload []byte) (msg *Message, err error) {
	transport, err := fw.NewTransportFromJSON(string(payload))
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

	msg, err = NewMessageFromRequest(request, transport.ReplyTo(), fw)
	if err != nil {
		return nil, err
	}

	return
}

// NewReplyFromTransportJSON creates a new Reply from a transport JSON
func (fw *Framework) NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error) {
	transport, err := fw.NewTransportFromJSON(string(payload))
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
	transport, err := fw.NewTransportFromJSON(string(payload))
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
func (fw *Framework) NewRequest(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
	switch version {
	case protocol.RequestV1:
		request, err = v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	default:
		err = fmt.Errorf("Do not know how to create a Request version %s", version)
	}

	return
}

// NewRequestFromMessage creates a new Request with the Message settings preloaded complying with a specific protocol version like protocol.RequestV1
func (fw *Framework) NewRequestFromMessage(version string, msg *Message) (req protocol.Request, err error) {
	if !(msg.Type() == "request" || msg.Type() == "direct_request") {
		err = fmt.Errorf("Cannot use `%s` message to construct a Request", msg.Type())
		return
	}

	req, err = fw.NewRequest(version, msg.Agent, msg.SenderID, msg.CallerID, msg.TTL, msg.RequestID, msg.Collective())
	if err != nil {
		return req, fmt.Errorf("Could not create a Request from a Message: %s", err)
	}

	req.SetMessage(msg.Payload)

	if msg.Filter == nil || msg.Filter.Empty() {
		req.NewFilter()
	} else {
		req.SetFilter(msg.Filter)
	}

	return
}

// NewReply creates a new Reply, the version will match that of the given request
func (fw *Framework) NewReply(request protocol.Request) (reply protocol.Reply, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		reply, err = v1.NewReply(request, fw.Config.Identity)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", request.Version())
	}

	return
}

// NewReplyFromMessage creates a new Reply with the Message settings preloaded complying with a specific protocol version like protocol.ReplyV1
func (fw *Framework) NewReplyFromMessage(version string, msg *Message) (rep protocol.Reply, err error) {
	if msg.Type() != "reply" {
		err = fmt.Errorf("Cannot use `%s` message to construct a Reply", msg.Type())
		return
	}

	if msg.Request == nil {
		err = fmt.Errorf("Cannot create a Reply from Messages without Requests")
		return
	}

	req, err := fw.NewRequestFromMessage(version, msg.Request)
	if err != nil {
		return
	}

	rep, err = fw.NewReply(req)
	rep.SetMessage(msg.Payload)

	return
}

// NewReplyFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (fw *Framework) NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error) {
	switch sr.Version() {
	case protocol.SecureReplyV1:
		reply, err = v1.NewReplyFromSecureReply(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewRequestFromSecureRequest creates a new Request from a SecureRequest, the version will match what is in the JSON payload
func (fw *Framework) NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error) {
	switch sr.Version() {
	case protocol.SecureRequestV1:
		request, err = v1.NewRequestFromSecureRequest(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewSecureReply creates a new SecureReply with the given Reply message as payload
func (fw *Framework) NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error) {
	switch reply.Version() {
	case protocol.ReplyV1:
		secure, err = v1.NewSecureReply(reply, fw.security)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply based on a Reply version %s", reply.Version())
	}

	return
}

// NewSecureReplyFromTransport creates a new SecureReply from the JSON payload of TransportMessage, the version SecureReply will be the same as the TransportMessage
func (fw *Framework) NewSecureReplyFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureReply, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureReplyFromTransport(message, fw.security, skipvalidate)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", message.Version())

	}

	return
}

// NewSecureRequest creates a new SecureRequest with the given Request message as payload
func (fw *Framework) NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		secure, err = v1.NewSecureRequest(request, fw.security)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a Request with version %s", request.Version())
	}

	return
}

// NewSecureRequestFromTransport creates a new SecureRequest from the JSON payload of TransportMessage, the version SecureRequest will be the same as the TransportMessage
func (fw *Framework) NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureRequestFromTransport(message, fw.security, skipvalidate)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a TransportMessage version %s", message.Version())
	}

	return
}

// NewTransportForSecureRequest creates a new TransportMessage with a SecureRequest as payload.  The Transport will be the same version as the SecureRequest
func (fw *Framework) NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error) {
	switch request.Version() {
	case protocol.SecureRequestV1:
		message, err = v1.NewTransportMessage(fw.Config.Identity)
		if err != nil {
			logrus.Errorf("Failed to create transport from secure request: %s", err)
			return
		}

		err = message.SetRequestData(request)
		if err != nil {
			logrus.Errorf("Failed to create transport from secure request: %s", err)
			return
		}

	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", request.Version())
	}

	return
}

// NewTransportForSecureReply creates a new TransportMessage with a SecureReply as payload.  The Transport will be the same version as the SecureRequest
func (fw *Framework) NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error) {
	switch reply.Version() {
	case protocol.SecureReplyV1:
		message, err = v1.NewTransportMessage(fw.Config.Identity)
		message.SetReplyData(reply)
	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", reply.Version())
	}

	return
}

// NewReplyTransportForMessage creates a new Transport message based on a Message and the request its a reply to
//
// The new transport message will have the same version as the request its based on
func (fw *Framework) NewReplyTransportForMessage(msg *Message, request protocol.Request) (protocol.TransportMessage, error) {
	reply, err := fw.NewReply(request)
	if err != nil {
		return nil, fmt.Errorf("Could not create Reply: %s", err)
	}

	reply.SetMessage(msg.Payload)

	sreply, err := fw.NewSecureReply(reply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Secure Reply: %s", err)
	}

	transport, err := fw.NewTransportForSecureReply(sreply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err)
	}

	protocol.CopyFederationData(request, transport)

	return transport, nil
}

// NewRequestTransportForMessage creates a new versioned Transport message based on a Message
func (fw *Framework) NewRequestTransportForMessage(msg *Message, version string) (protocol.TransportMessage, error) {
	req, err := fw.NewRequestFromMessage(version, msg)
	if err != nil {
		return nil, fmt.Errorf("Could not create Request: %s", err)
	}

	sr, err := fw.NewSecureRequest(req)
	if err != nil {
		return nil, fmt.Errorf("Could not create Secure Request: %s", err)
	}

	transport, err := fw.NewTransportForSecureRequest(sr)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err)
	}

	return transport, nil
}

// NewTransportMessage creates a new TransportMessage complying with a specific protocol version like protocol.TransportV1
func (fw *Framework) NewTransportMessage(version string) (message protocol.TransportMessage, err error) {
	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportMessage(fw.Config.Identity)
	default:
		err = fmt.Errorf("Do not know how to create a Transport version '%s'", version)
	}

	return
}

// NewTransportFromJSON creates a new TransportMessage from a JSON payload.  The version will match what is in the payload
func (fw *Framework) NewTransportFromJSON(data string) (message protocol.TransportMessage, err error) {
	version := gjson.Get(data, "protocol").String()

	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportFromJSON(data)
	default:
		err = fmt.Errorf("Do not know how to create a TransportMessage from an expected JSON format message with content: %s", data)
	}

	return
}
