// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package signers

import (
	"context"
	"fmt"

	aaac "github.com/choria-io/go-choria/client/aaa_signerclient"
	"github.com/choria-io/go-choria/inter"
)

// NewAAAServiceRPCSigner creates an AAA Signer that uses Choria RPC requests to the AAA Service
func NewAAAServiceRPCSigner(fw inter.Framework) *aaaServiceRPC {
	return &aaaServiceRPC{fw: fw}
}

type aaaServiceRPC struct {
	fw inter.Framework
}

func (s *aaaServiceRPC) Kind() string { return "AAA Service RPC" }

func (s *aaaServiceRPC) Sign(ctx context.Context, request []byte, cfg inter.RequestSignerConfig) ([]byte, error) {
	signer, err := aaac.New(s.fw)
	if err != nil {
		return nil, err
	}

	token, err := cfg.RemoteSignerToken()
	if err != nil {
		return nil, err
	}

	res, err := signer.OptionWorkers(1).Sign(string(request), string(token)).Do(ctx)
	if err != nil {
		return nil, err
	}

	if res.Stats().ResponsesCount() != 1 {
		return nil, fmt.Errorf("expected 1 response received %d", res.Stats().ResponsesCount())
	}

	var signed []byte

	res.EachOutput(func(r *aaac.SignOutput) {
		if !r.ResultDetails().OK() {
			err = fmt.Errorf("signing failed: %s", r.ResultDetails().StatusMessage())
		}

		signed = []byte(r.SecureRequest())
	})

	return signed, err
}
