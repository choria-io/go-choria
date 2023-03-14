// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
)

type Message struct {
	ID       string            `json:"id"`
	Subject  string            `json:"subject"`
	Payload  []byte            `json:"payload"`
	Reliable bool              `json:"reliable"`
	Priority uint              `json:"priority"`
	Created  time.Time         `json:"created"`
	TTL      float64           `json:"ttl"`
	MaxTries uint              `json:"max_tries"`
	Tries    uint              `json:"tries"`
	NextTry  time.Time         `json:"next_try"`
	Sender   string            `json:"sender"`
	Identity string            `json:"identity"`
	Sign     bool              `json:"sign"`
	Headers  map[string]string `json:"headers"`

	st StoreType
	sm any
}

func newMessage(sender string) *Message {
	return &Message{
		ID:       util.UniqueID(),
		Created:  time.Now(),
		MaxTries: 500,
		TTL:      defaultTTL.Seconds(),
		Sender:   sender,
	}
}

var (
	ErrMessageExpired     = errors.New("message has expired")
	ErrMessageMaxTries    = errors.New("message reached maximum tries")
	ErrSeedFileNotSet     = errors.New("seed file not set to sign message")
	ErrSeedFileNotFound   = errors.New("seed file not found")
	ErrSignatureFailed    = errors.New("could not calculate message signature")
	ErrReservedHeaderName = errors.New("headers may not start with 'choria' or 'nats'")
)

const (
	HdrNatsMsgId       = "Nats-Msg-Id"
	HdrChoriaPriority  = "Choria-Priority"
	HdrChoriaCreated   = "Choria-Created"
	HdrChoriaSender    = "Choria-Sender"
	HdrChoriaReliable  = "Choria-Reliable"
	HdrChoriaTries     = "Choria-Tries"
	HdrChoriaIdentity  = "Choria-Identity"
	HdrChoriaToken     = "Choria-Token"
	HdrChoriaSignature = "Choria-Signature"
	HdrChoriaPrefix    = "choria"
	HdrNatsPrefix      = "nats"
)

func (m *Message) Validate() error {
	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if len(m.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}

	if m.Priority > 4 {
		return fmt.Errorf("priority must be 0...4")
	}

	if m.Created.IsZero() {
		return fmt.Errorf("created time is zero")
	}

	if m.TTL == 0 {
		return fmt.Errorf("ttl may not be 0")
	}

	if m.TTL > maxTTL.Seconds() {
		return fmt.Errorf("ttl exceeds maximum %s", maxTTL)
	}

	if m.Sender == "" {
		return fmt.Errorf("sender is required")
	}

	if time.Since(m.Created) > (time.Duration(m.TTL) * time.Second) {
		return ErrMessageExpired
	}

	for k := range m.Headers {
		kl := strings.ToLower(k)
		if strings.HasPrefix(kl, HdrChoriaPrefix) || strings.HasPrefix(kl, HdrNatsPrefix) {
			return ErrReservedHeaderName
		}
	}

	return nil
}

func (m *Message) NatsMessage(prefix string, seed string, token string) (*nats.Msg, error) {
	err := m.Validate()
	if err != nil {
		return nil, err
	}

	if m.Tries >= m.MaxTries {
		return nil, ErrMessageMaxTries
	}

	msg := nats.NewMsg(prefix + m.Subject)

	for k, v := range m.Headers {
		msg.Header.Add(k, v)
	}

	msg.Header.Add(HdrNatsMsgId, m.ID)
	msg.Header.Add(HdrChoriaPriority, strconv.Itoa(int(m.Priority)))
	msg.Header.Add(HdrChoriaCreated, strconv.Itoa(int(m.Created.UnixNano())))
	msg.Header.Add(HdrChoriaSender, m.Sender)

	if m.Reliable {
		msg.Header.Add(HdrChoriaReliable, "1")
	}

	if m.Tries > 0 {
		msg.Header.Add(HdrChoriaTries, strconv.Itoa(int(m.Tries)))
	}

	if m.Identity != "" {
		msg.Header.Add(HdrChoriaIdentity, m.Identity)
	}

	msg.Data = m.Payload

	if m.Sign {
		if seed == "" {
			return nil, ErrSeedFileNotSet
		}

		if !util.FileExist(seed) {
			return nil, ErrSeedFileNotFound
		}

		sig, err := util.Ed25519SignWithSeedFile(seed, msg.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSignatureFailed, err)
		}

		if token != "" && util.FileExist(token) {
			t, err := os.ReadFile(token)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrSignatureFailed, err)
			}
			msg.Header.Add(HdrChoriaToken, string(t))
		}
		msg.Header.Add(HdrChoriaSignature, hex.EncodeToString(sig))
	}

	return msg, nil
}
