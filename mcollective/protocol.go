package mcollective

import (
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/protocol/v1"
	"github.com/tidwall/gjson"
)

// NewMessage creates a new Message associated with this Choria instance
func (c *Choria) NewMessage(payload string, agent string, collective string, msgType string, request *Message) (msg *Message, err error) {
	msg, err = NewMessage(payload, agent, collective, msgType, request, c)

	return
}

func (c *Choria) NewTransportFromMessage(msg *Message, request protocol.Request) (protocol.TransportMessage, error) {
	reply, err := c.NewReply(request)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	reply.SetMessage(msg.Payload)

	sreply, err := c.NewSecureReply(reply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	transport, err := c.NewTransportForSecureReply(sreply)
	if err != nil {
		return nil, fmt.Errorf("Could not create Transport: %s", err.Error())
	}

	return transport, nil
}

func (c *Choria) NewMessageFromTransportJSON(payload []byte) (msg *Message, err error) {
	transport, err := c.NewTransportFromJSON(string(payload))
	if err != nil {
		return nil, err
	}

	srequest, err := c.NewSecureRequestFromTransport(transport, false)
	if err != nil {
		return nil, err
	}

	request, err := c.NewRequestFromSecureRequest(srequest)
	if err != nil {
		return nil, err
	}

	msg, err = NewMessageFromRequest(request, transport.ReplyTo(), c)
	if err != nil {
		return nil, err
	}

	return
}

// NewRequest creates a new Request complying with a specific protocol version like protocol.RequestV1
func (c *Choria) NewRequest(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
	switch version {
	case protocol.RequestV1:
		request, err = v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	default:
		err = fmt.Errorf("Do not know how to create a Request version %s", version)
	}

	return
}

// NewRequestFromMessage creates a new Request with the Message settings preloaded complying with a specific protocol version like protocol.RequestV1
func (c *Choria) NewRequestFromMessage(version string, msg *Message) (req protocol.Request, err error) {
	if !(msg.Type() == "request" || msg.Type() == "direct_request") {
		err = fmt.Errorf("Cannot use `%s` message to construct a Request", msg.Type())
		return
	}

	req, err = c.NewRequest(version, msg.Agent, msg.SenderID, msg.CallerID, msg.TTL, msg.RequestID, msg.Collective())
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
func (c *Choria) NewReply(request protocol.Request) (reply protocol.Reply, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		reply, err = v1.NewReply(request, c.Certname())
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", request.Version())
	}

	return
}

// NewReplyFromMessage creates a new Reply with the Message settings preloaded complying with a specific protocol version like protocol.ReplyV1
func (c *Choria) NewReplyFromMessage(version string, msg *Message) (rep protocol.Reply, err error) {
	if msg.Type() != "reply" {
		err = fmt.Errorf("Cannot use `%s` message to construct a Reply", msg.Type())
		return
	}

	if msg.Request == nil {
		err = fmt.Errorf("Cannot create a Reply from Messages without Requests")
		return
	}

	req, err := c.NewRequestFromMessage(version, msg.Request)
	if err != nil {
		return
	}

	rep, err = c.NewReply(req)
	rep.SetMessage(msg.Base64Payload())

	return
}

// NewReplyFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (c *Choria) NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error) {
	switch sr.Version() {
	case protocol.SecureReplyV1:
		reply, err = v1.NewReplyFromSecureReply(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewRequestFromSecureReply creates a new Reply from the JSON payload of SecureReply, the version will match what is in the JSON payload
func (c *Choria) NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error) {
	switch sr.Version() {
	case protocol.SecureRequestV1:
		request, err = v1.NewRequestFromSecureRequest(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

// NewSecureReply creates a new SecureReply with the given Reply message as payload
func (c *Choria) NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error) {
	switch reply.Version() {
	case protocol.ReplyV1:
		secure, err = v1.NewSecureReply(reply)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply based on a Reply version %s", reply.Version())
	}

	return
}

// NewSecureReplyFromTransport creates a new SecureReply from the JSON payload of TransportMessage, the version SecureReply will be the same as the TransportMessage
func (c *Choria) NewSecureReplyFromTransport(message protocol.TransportMessage) (secure protocol.SecureReply, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureReplyFromTransport(message)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", message.Version())

	}

	return
}

// NewSecureRequest creates a new SecureRequest with the given Request message as payload
func (c *Choria) NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		var pub, pri string

		pub, err = c.ClientPublicCert()
		if err != nil {
			return secure, err
		}

		pri, err = c.ClientPrivateKey()
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
func (c *Choria) NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureRequestFromTransport(message, skipvalidate)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a TransportMessage version %s", message.Version())
	}

	return
}

// NewTransportForSecureRequest creates a new TransportMessage with a SecureRequest as payload.  The Transport will be the same version as the SecureRequest
func (c *Choria) NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error) {
	switch request.Version() {
	case protocol.SecureRequestV1:
		message, err = v1.NewTransportMessage(c.Certname())
		message.SetRequestData(request)
	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", request.Version())
	}

	return
}

// NewTransportForSecureReply creates a new TransportMessage with a SecureReply as payload.  The Transport will be the same version as the SecureRequest
func (c *Choria) NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error) {
	switch reply.Version() {
	case protocol.SecureReplyV1:
		message, err = v1.NewTransportMessage(c.Certname())
		message.SetReplyData(reply)
	default:
		err = fmt.Errorf("Do not know how to create a Transport message for SecureRequest version %s", reply.Version())
	}

	return
}

// NewTransportMessage creates a new TransportMessage complying with a specific protocol version like protocol.TransportV1
func (c *Choria) NewTransportMessage(version string) (message protocol.TransportMessage, err error) {
	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportMessage(c.Certname())
	default:
		err = fmt.Errorf("Do not know how to create a Transport version '%s'", version)
	}

	return
}

// NewTransportFromJSON creates a new TransportMessage from a JSON payload.  The version will match what is in the payload
func (c *Choria) NewTransportFromJSON(data string) (message protocol.TransportMessage, err error) {
	version := gjson.Get(data, "protocol").String()

	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportFromJSON(data)
	default:
		err = fmt.Errorf("Do not know how to create a TransportMessage from an expected JSON format message with content: %s", data)
	}

	return
}
