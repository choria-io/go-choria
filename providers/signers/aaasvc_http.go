// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package signers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/choria-io/go-choria/inter"
)

// NewAAAServiceHTTPSigner creates an AAA Signer that uses HTTP requests to the AAA Service
func NewAAAServiceHTTPSigner() *aaaServiceHTTP {
	return &aaaServiceHTTP{}
}

type aaaServiceHTTP struct{}

type httpSigningRequest struct {
	Token   string `json:"token"`
	Request []byte `json:"request"`
}

type httpSigningReply struct {
	Signed []byte `json:"secure_request"`
	Error  string `json:"error"`
}

func (s *aaaServiceHTTP) Kind() string { return "AAA Service HTTP" }

func (s *aaaServiceHTTP) Sign(_ context.Context, request []byte, cfg inter.RequestSignerConfig) ([]byte, error) {
	signer, err := cfg.RemoteSignerURL()
	if err != nil {
		return nil, fmt.Errorf("remote signing URL not configured")
	}

	token, err := cfg.RemoteSignerToken()
	if err != nil {
		return nil, err
	}

	req := &httpSigningRequest{Request: request}
	req.Token = string(token)

	client := &http.Client{}
	if signer.Scheme == "https" {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// While this might appear alarming it's expected that the clients
			// in this situation will not have any Choria CA issued certificates
			// and so wish to use a remote signer - the certificate management woes
			// being one of the main reasons for centralized AAA.
			//
			// So there is no realistic way to verify these requests especially in the
			// event that these signers run on private IPs and such as would be typical
			// so while we do this big No No of disabling verify here it really is the
			// only thing that make sense.
			InsecureSkipVerify: true,
		}}
	}

	jreq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not encode request: %s", err)
	}

	resp, err := client.Post(signer.String(), "application/json", bytes.NewBuffer(jreq))
	if err != nil {
		return nil, fmt.Errorf("could not perform signing request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not perform remote signing request: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read signing response: %s", err)
	}

	signerResp := &httpSigningReply{}
	err = json.Unmarshal(body, signerResp)
	if err != nil {
		return nil, fmt.Errorf("could not parse signer response: %s", err)
	}

	if signerResp.Error != "" {
		return nil, fmt.Errorf("error from remote signer: %s", signerResp.Error)
	}

	return signerResp.Signed, nil

}
