// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

// ConnectorMessage is received from middleware
type ConnectorMessage interface {
	Subject() string
	Reply() string
	Data() []byte

	// Msg is the middleware specific message like *nats.Msg
	Msg() interface{}
}
