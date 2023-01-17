// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"

	"github.com/choria-io/go-choria/inter"
)

type funcSigner struct {
	fn func(context.Context, []byte, inter.RequestSignerConfig) ([]byte, error)
}

func NewFuncSigner(fn func(context.Context, []byte, inter.RequestSignerConfig) ([]byte, error)) *funcSigner {
	return &funcSigner{fn: fn}
}

func (s *funcSigner) Kind() string { return "Integration Signer" }
func (s *funcSigner) Sign(ctx context.Context, request []byte, cfg inter.RequestSignerConfig) ([]byte, error) {
	return s.fn(ctx, request, cfg)
}
