package choria

import (
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/protocol/v1"
	"github.com/tidwall/gjson"
)

// NewMessage creates a new Message associated with this Choria instance
func (self *Framework) NewMessage(payload string, agent string, collective string, msgType string, request *Message) (msg *Message, err error) {
	msg, err = NewMessage(payload, agent, collective, msgType, request, self)

	return
}

func (self *Framework) NewRequestMessageFromTransportJSON(payload []byte) (msg *Message, err error) {
	transport, err := self.NewTransportFromJSON(string(payload))
	if err != nil {
		return nil, err
	}

	srequest, err := self.NewSecureRequestFromTransport(transport, false)
	if err != nil {
		return nil, err
	}

	request, err := self.NewRequestFromSecureRequest(srequest)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, request)

	msg, err = NewMessageFromRequest(request, transport.ReplyTo(), self)
	if err != nil {
		return nil, err
	}

	return
}

// NewReplyFromTransportJSON creates a new Reply from a transport JSON
func (self *Framework) NewReplyFromTransportJSON(payload []byte) (msg protocol.Reply, err error) {
	transport, err := self.NewTransportFromJSON(string(payload))
	if err != nil {
		return nil, err
	}

	sreply, err := self.NewSecureReplyFromTransport(transport)
	if err != nil {
		return nil, err
	}

	reply, err := self.NewReplyFromSecureReply(sreply)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, reply)

	return reply, nil
}

// NewRequestFromTransportJSON creates a new Request from transport JSON
func (self *Framework) NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error) {
	transport, err := self.NewTransportFromJSON(string(payload))
	if err != nil {
		return nil, err
	}

	sreq, err := self.NewSecureRequestFromTransport(transport, skipvalidate)
	if err != nil {
		return nil, err
	}

	req, err := self.NewRequestFromSecureRequest(sreq)
	if err != nil {
		return nil, err
	}

	protocol.CopyFederationData(transport, req)

	return req, nil
}

// NewRequest creates a new Request complying with a specific protocol version like protocol.RequestV1
func (self *Framework) NewRequest(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
	switch version {
	case protocol.RequestV1:
		request, err = v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	default:
		err = fmt.Errorf("Do not know how to create a Request version %s", version)
	}

	return
}

// NewRequestFromMessage creates a new Request with the Message settings preloaded complying with a specific protocol version like protocol.RequestV1
func (self *Framework) NewRequestFromMessage(version string, msg *Message) (req protocol.Request, err error) {
	if !(msg.Type() == "request" || msg.Type() == "direct_request") {
		err = fmt.Errorf("Cannot use `%s` message to construct a Request", msg.Type())
		return
	}

	req, err = self.NewRequest(version, msg.Agent, msg.SenderID, msg.CallerID, msg.TTL, msg.RequestID, msg.Collective())
	if err != nil {
		return req, fmt.Errorf("Could not create a Request from a Message: %s", err.Error())
	}

	req.SetMessage(msg.Base64Payload())

	if msg.Filter == nil || msg.Filter.Empty() {
		req.NewFilter()
	} else {
		req.SetFilter(msg.Filter)
	}

	return
}

// NewReply creates a new Reply, the version will match that of the given request
func (self *Framework) NewReply(request protocol.Request) (reply protocol.Reply, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		reply, err = v1.NewReply(request, self.Config.Identity)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", request.Version())
	}

	return
}

// NewReplyFromMessage creates a new Reply with the Message settings preloaded complying with a specific protocol version like protocol.ReplyV1
func (self *Framework) NewReplyFromMessage(version string, msg *Message) (rep protocol.Reply, err error) {
	if msg.Type() != "reply" {
		err = fmt.Errorf("Cannot use `%s` message to construct a Reply", msg.Type())
		return
	}

	if msg.Request == nil {
		err = fmt.Errorf("Cannot create a Reply from Messages without Requests")
		return
	}

	req, err := self.NewRequestFromMessage(version, msg.Request)
	if err != nil {
		return
	}

	rep, err = self.NewReply(req)
	rep.SetMessage(msg.Base64Payload())

	return
}

// NewReplyFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (self *Framework) NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error) {
	switch sr.Version() {
	case protocol.SecureReplyV1:
		reply, err = v1.NewReplyFromSecureReply(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewRequestFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (self *Framework) NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error) {
	switch sr.Version() {
	case protocol.SecureRequestV1:
		request, err = v1.NewRequestFromSecureRequest(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewSecureReply creates a new SecureReply with the given Reply message as payload
func (self *Framework) NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error) {
	switch reply.Version() {
	case protocol.ReplyV1:
		secure, err = v1.NewSecureReply(reply)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply based on a Reply version %s", reply.Version())
	}

	return
}

// NewSecureReplyFromTransport creates a new SecureReply from the JSON payload of TransportMessage, the version SecureReply will be the same as the TransportMessage
func (self *Framework) NewSecureReplyFromTransport(message protocol.TransportMessage) (secure protocol.SecureReply, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureReplyFromTransport(message)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", message.Version())

	}

	return
}

// NewSecureRequest creates a new SecureRequest with the given Request message as payload
func (self *Framework) NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		var pub, pri string

		pub, err = self.ClientPublicCert()
		if err != nil {
			return secure, err
		}

		pri, err = self.ClientPrivateKey()
		if err != nil {
			return secure, err
		}

		secure, err = v1.NewSecureRequest(request, pub, pri)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a Request with version %s", request.Version())
	}

	return
}

// NewSecureRequestFromTransport creates a new SecureRequest from the JSON payload of TransportMessage, the version SecureRequest will be the same as the TransportMessage
func (self *Framework) NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		var ca string
		var cache string

		ca, err = self.CAPath()
		if err != nil {
			return
		}

		cache, err = self.ClientCertCacheDir()
		if err != nil {
			return
		}

		secure, err = v1.NewSecureRequestFromTransport(message, ca, cache, self.Config.Choria.CertnameWhitelist, self.Config.Choria.PrivilegedUsers, skipvalidate)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a TransportMessage version %s", message.Version())
	}

	return
}

// NewTransportForSecureRequest creates a new TransportMessage with a SecureRequest as payload.  The Transport will be the same version as the SecureRequest
func (self *Framework) NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error) {
	switch request.Version() {
	case protocol.SecureRequestV1:
		message, err = v1.NewTransportMessage(self.Config.Identity)
		message.SetRequestData(request)
	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", request.Version())
	}

	return
}

// NewTransportForSecureReply creates a new TransportMessage with a SecureReply as payload.  The Transport will be the same version as the SecureRequest
func (self *Framework) NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error) {
	switch reply.Version() {
	case protocol.SecureReplyV1:
		message, err = v1.NewTransportMessage(self.Config.Identity)
		message.SetReplyData(reply)
	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", reply.Version())
	}

	return
}

// NewReplyTransportForMessage creates a new Transport message based on a Message and the request its a reply to
//
// The new transport message will have the same version as the request its based on
func (self *Framework) NewReplyTransportForMessage(msg *Message, request protocol.Request) (protocol.TransportMessage, error) {
	reply, err := self.NewReply(request)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	reply.SetMessage(msg.Payload)

	sreply, err := self.NewSecureReply(reply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	transport, err := self.NewTransportForSecureReply(sreply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	protocol.CopyFederationData(request, transport)

	return transport, nil
}

// NewRequestTransportForMessage creates a new versioned Transport message based on a Message
func (self *Framework) NewRequestTransportForMessage(msg *Message, version string) (protocol.TransportMessage, error) {
	req, err := self.NewRequestFromMessage(version, msg)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	sr, err := self.NewSecureRequest(req)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	transport, err := self.NewTransportForSecureRequest(sr)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	return transport, nil
}

// NewTransportMessage creates a new TransportMessage complying with a specific protocol version like protocol.TransportV1
func (self *Framework) NewTransportMessage(version string) (message protocol.TransportMessage, err error) {
	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportMessage(self.Config.Identity)
	default:
		err = fmt.Errorf("Do not know how to create a Transport version '%s'", version)
	}

	return
}

// NewTransportFromJSON creates a new TransportMessage from a JSON payload.  The version will match what is in the payload
func (self *Framework) NewTransportFromJSON(data string) (message protocol.TransportMessage, err error) {
	version := gjson.Get(data, "protocol").String()

	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportFromJSON(data)
	default:
		err = fmt.Errorf("Do not know how to create a TransportMessage from an expected JSON format message with content: %s", data)
	}

	return
}
