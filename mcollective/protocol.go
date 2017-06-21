package mcollective

import (
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/protocol/v1"
	"github.com/tidwall/gjson"
)

func (c *Choria) NewMessage(payload string, agent string, collective string, msgType string, request *Message) (msg *Message, err error) {
	msg, err = NewMessage(payload, agent, collective, msgType, request, c)

	return
}

func (c *Choria) NewRequest(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error) {
	switch version {
	case protocol.RequestV1:
		request, err = v1.NewRequest(agent, senderid, callerid, ttl, requestid, collective)
	default:
		err = fmt.Errorf("Do not know how to create a Request version %s", version)
	}

	return
}

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

func (c *Choria) NewReply(version string, request protocol.Request) (reply protocol.Reply, err error) {
	switch version {
	case protocol.ReplyV1:
		reply, err = v1.NewReply(request)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", version)
	}

	return
}

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

	rep, err = c.NewReply(version, req)
	rep.SetMessage(msg.Base64Payload())

	return
}

func (c *Choria) NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error) {
	switch sr.Version() {
	case protocol.SecureReplyV1:
		reply, err = v1.NewReplyFromSecureReply(sr)
	default:
		err = fmt.Errorf("Do not know how to create a Reply version %s", sr.Version())
	}

	return
}

func (c *Choria) NewSecureReply(version string, reply protocol.Reply) (secure protocol.SecureReply, err error) {
	switch version {
	case protocol.SecureReplyV1:
		secure, err = v1.NewSecureReply(reply)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", version)
	}

	return
}

func (c *Choria) NewSecureReplyFromTransport(message protocol.TransportMessage) (secure protocol.SecureReply, err error) {
	// TODO: Check its actually a reply in the data

	switch message.Version() {
	case protocol.TransportV1:
		secure, err = v1.NewSecureReplyFromTransport(message)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", message.Version())

	}

	return
}

func (c *Choria) NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error) {
	switch request.Version() {
	case protocol.RequestV1:
		pub, err := c.ClientPublicCert()
		if err != nil {
			return nil, err
		}

		pri, err := c.ClientPrivateKey()
		if err != nil {
			return nil, err
		}

		secure, err = v1.NewSecureRequest(request, pub, pri)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply from a Request with version %s", request.Version())
	}

	return
}

func (c *Choria) NewSecureRequestFromTransport(message protocol.TransportMessage) (secure protocol.SecureRequest, err error) {
	// TODO: Check its actually a request in the data

	switch message.Version() {
	case protocol.SecureRequestV1:
		secure, err = v1.NewSecureRequestFromTransport(message)
	default:
		err = fmt.Errorf("Do not know how to create a SecureReply version %s", message.Version())
	}

	return
}

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

func (c *Choria) NewTransportMessage(version string) (message protocol.TransportMessage, err error) {
	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportMessage(c.Certname())
	default:
		err = fmt.Errorf("Do not know how to create a Transport version %s", version)
	}

	return
}

func (c *Choria) NewTransportFromJSON(data string) (message protocol.TransportMessage, err error) {
	version := gjson.Get(data, "protocol").String()

	switch version {
	case protocol.TransportV1:
		message, err = v1.NewTransportFromJSON(data)
	default:
		err = fmt.Errorf("Do not know how to create a Transport version %s", version)
	}

	return
}
