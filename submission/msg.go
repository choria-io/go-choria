// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
)

type Message struct {
	ID       string    `json:"id"`
	Subject  string    `json:"subject"`
	Payload  []byte    `json:"payload"`
	Reliable bool      `json:"reliable"`
	Priority uint      `json:"priority"`
	Created  time.Time `json:"created"`
	TTL      float64   `json:"ttl"`
	MaxTries uint      `json:"max_tries"`
	Tries    uint      `json:"tries"`
	NextTry  time.Time `json:"next_try"`
	Sender   string    `json:"sender"`
	Identity string    `json:"identity"`

	st StoreType
	sm interface{}
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
	ErrMessageExpired  = errors.New("message has expired")
	ErrMessageMaxTries = errors.New("message reached maximum tries")
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

	return nil
}

func (m *Message) NatsMessage(prefix string) (*nats.Msg, error) {
	err := m.Validate()
	if err != nil {
		return nil, err
	}

	if m.Tries >= m.MaxTries {
		return nil, ErrMessageMaxTries
	}

	msg := nats.NewMsg(prefix + m.Subject)
	msg.Header.Add("Nats-Msg-Id", m.ID)
	msg.Header.Add("Choria-Priority", strconv.Itoa(int(m.Priority)))
	msg.Header.Add("Choria-Created", strconv.Itoa(int(m.Created.UnixNano())))
	msg.Header.Add("Choria-Sender", m.Sender)

	if m.Reliable {
		msg.Header.Add("Choria-Reliable", "1")
	}

	if m.Tries > 0 {
		msg.Header.Add("Choria-Tries", strconv.Itoa(int(m.Tries)))
	}

	if m.Identity != "" {
		msg.Header.Add("Choria-Identity", m.Identity)
	}

	msg.Data = m.Payload

	return msg, nil
}
