// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
)

// NewRequest creates a choria:request:1
func NewRequest(agent string, senderid string, callerid string, ttl int, requestid string, collective string) (req protocol.Request, err error) {
	req = &Request{
		Protocol: protocol.RequestV1,
		Envelope: &RequestEnvelope{
			SenderID:  senderid,
			TTL:       ttl,
			RequestID: requestid,
			Time:      time.Now().Unix(),
		},
	}

	req.SetCollective(collective)
	req.SetAgent(agent)
	req.SetCallerID(callerid)
	req.SetFilter(protocol.NewFilter())

	return req, nil
}

// NewReply creates a choria:reply:1 based on a previous Request
func NewReply(request protocol.Request, certname string) (rep protocol.Reply, err error) {
	if request.Version() != protocol.RequestV1 {
		return nil, fmt.Errorf("cannot create a version 1 Reply from a %s request", request.Version())
	}

	rep = &Reply{
		Protocol: protocol.ReplyV1,
		Envelope: &ReplyEnvelope{
			RequestID: request.RequestID(),
			SenderID:  certname,
			Agent:     request.Agent(),
			Time:      time.Now().Unix(),
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

// NewReplyFromSecureReply create a choria:reply:1 based on the data contained in a SecureReply
func NewReplyFromSecureReply(sr protocol.SecureReply) (rep protocol.Reply, err error) {
	if sr.Version() != protocol.SecureReplyV1 {
		return nil, fmt.Errorf("cannot create a version 1 SecureReply from a %s SecureReply", sr.Version())
	}

	rep = &Reply{
		Protocol: protocol.ReplyV1,
		Envelope: &ReplyEnvelope{},
	}

	err = rep.IsValidJSON(sr.Message())
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the SecureReply is not a valid Reply message: %s", err)
	}

	err = json.Unmarshal(sr.Message(), rep)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data from Secure Reply: %s", err)
	}

	return rep, nil
}

// NewRequestFromSecureRequest creates a choria::request:1 based on the data contained in a SecureRequest
func NewRequestFromSecureRequest(sr protocol.SecureRequest) (protocol.Request, error) {
	if sr.Version() != protocol.SecureRequestV1 {
		return nil, fmt.Errorf("cannot create a version 1 SecureRequest from a %s SecureRequest", sr.Version())
	}

	req := &Request{
		Protocol: protocol.RequestV1,
		Envelope: &RequestEnvelope{},
	}

	err := req.IsValidJSON(sr.Message())
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the SecureRequest is not a valid Request message: %s", err)
	}

	err = json.Unmarshal(sr.Message(), req)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data from Secure Request: %s", err)
	}

	return req, nil
}

// NewSecureReply creates a choria:secure:reply:1
func NewSecureReply(reply protocol.Reply, security inter.SecurityProvider) (secure protocol.SecureReply, err error) {
	secure = &SecureReply{
		Protocol: protocol.SecureReplyV1,
		security: security,
	}

	err = secure.SetMessage(reply)
	if err != nil {
		return nil, fmt.Errorf("could not set message on SecureReply structure: %s", err)
	}

	return secure, nil
}

// NewSecureReplyFromTransport creates a new choria:secure:reply:1 from the data contained in a Transport message
func NewSecureReplyFromTransport(message protocol.TransportMessage, security inter.SecurityProvider, skipvalidate bool) (secure protocol.SecureReply, err error) {
	secure = &SecureReply{
		Protocol: protocol.SecureReplyV1,
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return nil, err
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the TransportMessage is not a valid SecureReply message: %s", err)
	}

	err = json.Unmarshal(data, &secure)
	if err != nil {
		return nil, err
	}

	if !skipvalidate {
		if !secure.Valid() {
			return nil, errors.New("SecureReply message created from the Transport Message is not valid")
		}
	}

	return secure, nil
}

// NewSecureRequest creates a choria:secure:request:1
func NewSecureRequest(request protocol.Request, security inter.SecurityProvider) (secure protocol.SecureRequest, err error) {
	pub := []byte("insecure")

	if protocol.IsSecure() && !protocol.IsRemoteSignerAgent(request.Agent()) {
		pub, err = security.PublicCertBytes()
		if err != nil {
			// registration when doing anon tls might not have a certificate - so we allow that to go unsigned
			if protocol.IsRegistrationAgent(request.Agent()) {
				pub = []byte("insecure registration")
			} else {
				return nil, fmt.Errorf("could not retrieve Public Certificate from the security subsystem: %s", err)
			}
		}
	}

	secure = &SecureRequest{
		Protocol:          protocol.SecureRequestV1,
		PublicCertificate: string(pub),
		security:          security,
	}

	err = secure.SetMessage(request)
	if err != nil {
		return nil, fmt.Errorf("could not set message SecureRequest structure: %s", err)
	}

	return secure, nil
}

// NewRemoteSignedSecureRequest is a NewSecureRequest that delegates the signing to a remote signer like aaasvc
func NewRemoteSignedSecureRequest(request protocol.Request, security inter.SecurityProvider) (secure protocol.SecureRequest, err error) {
	// no need for remote stuff, we don't do any signing or certs,
	// additionally the service hosting the remote signing service isnt
	// secured by choria protocol since at calling time the client does
	// not have a cert etc, but the request expects a signed JWT so that
	// provides the security of that request
	if !protocol.IsSecure() || protocol.IsRemoteSignerAgent(request.Agent()) {
		return NewSecureRequest(request, security)
	}

	reqj, err := request.JSON()
	if err != nil {
		return nil, err
	}

	secj, err := security.RemoteSignRequest(context.Background(), []byte(reqj))
	if err != nil {
		return nil, err
	}

	secure = &SecureRequest{
		Protocol: protocol.SecureRequestV1,
		security: security,
	}

	err = json.Unmarshal(secj, &secure)
	if err != nil {
		return nil, fmt.Errorf("could not parse signed request: %s", err)
	}

	return secure, nil
}

// NewSecureRequestFromTransport creates a new choria:secure:request:1 from the data contained in a Transport message
func NewSecureRequestFromTransport(message protocol.TransportMessage, security inter.SecurityProvider, skipvalidate bool) (secure protocol.SecureRequest, err error) {
	secure = &SecureRequest{
		security: security,
	}

	data, err := message.Message()
	if err != nil {
		return
	}

	err = secure.IsValidJSON(data)
	if err != nil {
		return nil, fmt.Errorf("the JSON body from the TransportMessage is not a valid SecureRequest message: %s", err)
	}

	err = json.Unmarshal(data, &secure)
	if err != nil {
		return nil, err
	}

	if !skipvalidate {
		if !secure.Valid() {
			return nil, fmt.Errorf("SecureRequest message created from the Transport Message did not pass security validation")
		}
	}

	return secure, nil
}

// NewTransportMessage creates a choria:transport:1
func NewTransportMessage(certname string) (message protocol.TransportMessage, err error) {
	message = &TransportMessage{
		Protocol: protocol.TransportV1,
		Headers:  &TransportHeaders{},
	}

	message.SetSender(certname)

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
