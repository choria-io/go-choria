package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/choria-io/go-protocol/protocol"
)

// NewRequest creates a choria:request:1
func NewRequest(agent string, senderid string, callerid string, ttl int, requestid string, collective string) (req protocol.Request, err error) {
	req = &request{
		Protocol: protocol.RequestV1,
		Envelope: &requestEnvelope{
			SenderID:  senderid,
			TTL:       ttl,
			RequestID: requestid,
			Time:      time.Now().UTC().Unix(),
		},
	}

	req.SetCollective(collective)
	req.SetAgent(agent)
	req.SetCallerID(callerid)
	req.SetFilter(protocol.NewFilter())

	return
}

// NewReply creates a choria:reply:1 based on a previous Request
func NewReply(request protocol.Request, certname string) (rep protocol.Reply, err error) {
	if request.Version() != protocol.RequestV1 {
		err = fmt.Errorf("Cannot create a version 1 Reply from a %s request", request.Version())
		return
	}

	rep = &reply{
		Protocol: protocol.ReplyV1,
		Envelope: &replyEnvelope{
			RequestID: request.RequestID(),
			SenderID:  certname,
			Agent:     request.Agent(),
			Time:      time.Now().UTC().Unix(),
		},
	}

	protocol.CopyFederationData(request, rep)

	j, err := request.JSON()
	if err != nil {
		err = fmt.Errorf("Could not turn Request %s into a JSON document: %s", request.RequestID(), err)
		return
	}

	rep.SetMessage(j)

	return
}

// NewReplyFromSecureReply create a choria:reply:1 based on the data contained in a SecureReply
func NewReplyFromSecureReply(sr protocol.SecureReply) (rep protocol.Reply, err error) {
	if sr.Version() != protocol.SecureReplyV1 {
		err = fmt.Errorf("Cannot create a version 1 SecureReply from a %s SecureReply", sr.Version())
		return
	}

	r := &reply{
		Protocol: protocol.ReplyV1,
		Envelope: &replyEnvelope{},
	}

	err = r.IsValidJSON(sr.Message())
	if err != nil {
		err = fmt.Errorf("The JSON body from the SecureReply is not a valid Reply message: %s", err)
		return
	}

	err = json.Unmarshal([]byte(sr.Message()), r)
	if err != nil {
		err = fmt.Errorf("Could not parse JSON data from Secure Reply: %s", err)
		return
	}

	rep = r

	return
}

// NewRequestFromSecureRequest creates a choria::request:1 based on the data contained in a SecureRequest
func NewRequestFromSecureRequest(sr protocol.SecureRequest) (req protocol.Request, err error) {
	if sr.Version() != protocol.SecureRequestV1 {
		err = fmt.Errorf("Cannot create a version 1 SecureRequest from a %s SecureRequest", sr.Version())
		return
	}

	r := &request{
		Protocol: protocol.RequestV1,
		Envelope: &requestEnvelope{},
	}

	err = r.IsValidJSON(sr.Message())
	if err != nil {
		err = fmt.Errorf("The JSON body from the SecureRequest is not a valid Request message: %s", err)
		return
	}

	err = json.Unmarshal([]byte(sr.Message()), r)
	if err != nil {
		err = fmt.Errorf("Could not parse JSON data from Secure Request: %s", err)
		return
	}

	req = r

	return
}

// NewSecureReply creates a choria:secure:reply:1
func NewSecureReply(reply protocol.Reply, security SecurityProvider) (secure protocol.SecureReply, err error) {
	secure = &secureReply{
		Protocol: protocol.SecureReplyV1,
		security: security,
	}

	err = secure.SetMessage(reply)
	if err != nil {
		err = fmt.Errorf("Could not set message on SecureReply structure: %s", err)
	}

	return
}

// NewSecureReplyFromTransport creates a new choria:secure:reply:1 from the data contained in a Transport message
func NewSecureReplyFromTransport(message protocol.TransportMessage, security SecurityProvider, skipvalidate bool) (secure protocol.SecureReply, err error) {
	secure = &secureReply{
		Protocol: protocol.SecureReplyV1,
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		err = fmt.Errorf("The JSON body from the TransportMessage is not a valid SecureReply message: %s", err)
		return
	}

	err = json.Unmarshal([]byte(data), &secure)
	if err != nil {
		return
	}

	if !skipvalidate {
		if !secure.Valid() {
			err = errors.New("SecureReply message created from the Transport Message is not valid: %s")
		}
	}

	return
}

// NewSecureRequest creates a choria:secure:request:1
func NewSecureRequest(request protocol.Request, security SecurityProvider) (secure protocol.SecureRequest, err error) {
	pub := []byte("insecure")

	if protocol.IsSecure() {
		pub, err = security.PublicCertTXT()
		if err != nil {
			err = fmt.Errorf("could not retrieve Public Certificate from the security subsystem: %s", err)
			return
		}
	}

	secure = &secureRequest{
		Protocol:          protocol.SecureRequestV1,
		PublicCertificate: string(pub),
		security:          security,
	}

	err = secure.SetMessage(request)
	if err != nil {
		err = fmt.Errorf("could not set message SecureRequest structure: %s", err)
	}

	return
}

// NewSecureRequestFromTransport creates a new choria:secure:request:1 from the data contained in a Transport message
func NewSecureRequestFromTransport(message protocol.TransportMessage, security SecurityProvider, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	secure = &secureRequest{
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		err = fmt.Errorf("The JSON body from the TransportMessage is not a valid SecureRequest message: %s", err)
		return
	}

	err = json.Unmarshal([]byte(data), &secure)
	if err != nil {
		return
	}

	if !skipvalidate {
		if !secure.Valid() {
			err = fmt.Errorf("SecureRequest message created from the Transport Message did not pass security validation")
		}
	}

	return
}

// NewTransportMessage creates a choria:transport:1
func NewTransportMessage(certname string) (message protocol.TransportMessage, err error) {
	message = &transportMessage{
		Protocol: protocol.TransportV1,
		Headers:  &transportHeaders{},
	}

	message.SetSender(certname)

	return
}

// NewTransportFromJSON creates a new TransportMessage from JSON
func NewTransportFromJSON(data string) (message protocol.TransportMessage, err error) {
	msg := &transportMessage{
		Headers: &transportHeaders{},
	}

	err = msg.IsValidJSON(data)
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return
	}

	message = msg

	return
}
