package signers

import (
	"context"
	"fmt"

	aaac "github.com/choria-io/go-choria/client/aaa_signerclient"
	"github.com/choria-io/go-choria/inter"
)

// NewAAAServiceRPCSigner creates an AAA Signer that uses Choria RPC requests to the AAA Service
func NewAAAServiceRPCSigner(fw aaac.ChoriaFramework) *aaaServiceRPC {
	return &aaaServiceRPC{fw: fw}
}

type aaaServiceRPC struct {
	fw aaac.ChoriaFramework
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

	res, err := signer.Sign(string(request), string(token)).Do(ctx)
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
