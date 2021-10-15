// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package data

// RegistrationItem contains a single registration message
type RegistrationItem struct {
	// Data is the raw data to publish
	Data []byte

	// Destination let you set custom NATS targets, when this is not set
	// the TargetAgent will be used to create a normal agent target
	Destination string

	// TargetAgent lets you pick where to send the data as a request
	TargetAgent string
}
