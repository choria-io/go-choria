// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

/*
Package lifecycle provides events that services in the Choria eco system
emit during startup, shutdown, provisioning and general running.

These events can be used by other tools to react to events or monitor the
running of a Chroia network.

A library to view the events received from the network and one to create
a running tally of the count and versions of nodes on your network.
*/
package lifecycle

import (
	"errors"
	"fmt"
	"sort"

	"github.com/choria-io/go-choria/inter"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tidwall/gjson"
)

// Type is a type of event this system supports
type Type int

const (
	// Unknown is for when a Type string was passed in that doesn't match what was expected
	Unknown Type = iota - 1

	// Startup is an event components can publish when they start
	Startup

	// Shutdown is an event components can publish when they shutdown
	Shutdown

	// Provisioned is an event components can publish post provisioning
	Provisioned

	// Alive is an event components can publish to indicate they are still alive
	Alive

	// Governor is an event components can publish while interacting with a Governor
	Governor
)

//lint:ignore U1000 #1768 support for external clients
func (t Type) String() string {
	switch t {
	case Startup:
		return "Startup"
	case Shutdown:
		return "Startup"
	case Provisioned:
		return "Provisioned"
	case Alive:
		return "Alive"
	case Governor:
		return "Governor"
	default:
		return "Unknown"
	}
}

// Format is the event format used for transporting events
type Format int

const (
	// UnknownFormat is an unknown format message
	UnknownFormat Format = iota

	// CloudEventV1Format is a classical Choria lifecycle event carried within a version 1.0 CloudEvent
	CloudEventV1Format
)

//lint:ignore U1000 #1768 support for external clients
func (f Format) String() string {
	switch f {
	case UnknownFormat:
		return "UnknownFormat"
	case CloudEventV1Format:
		return "CloudEventV1Format"
	default:
		return "UnknownFormat"
	}
}

var eventTypes = make(map[string]Type)
var eventJSONParsers = make(map[Type]func([]byte) (Event, error))
var eventFactories = make(map[Type]func(...Option) Event)

// New creates a new event
func New(t Type, opts ...Option) (Event, error) {
	factory, ok := eventFactories[t]
	if !ok {
		return nil, errors.New("unknown event type")
	}

	return factory(opts...), nil
}

// EventTypeNames produce a list of valid event type names
func EventTypeNames() []string {
	names := []string{}

	for k := range eventTypes {
		names = append(names, k)
	}

	sort.Strings(names)

	return names
}

// EventFormatFromJSON inspects the JSON data and tries to determine the format from it's content
func EventFormatFromJSON(j []byte) Format {
	specversion := gjson.GetBytes(j, "specversion")
	source := gjson.GetBytes(j, "source")

	if specversion.Exists() && source.Exists() {
		if specversion.String() == "1.0" && source.String() == "io.choria.lifecycle" {
			return CloudEventV1Format
		}
	}

	return UnknownFormat
}

// NewFromJSON creates an event from the event JSON
func NewFromJSON(j []byte) (event Event, err error) {
	format := EventFormatFromJSON(j)

	switch format {
	case CloudEventV1Format:
		event, err = cloudeventV1FormatNewFromJSON(j)
	default:
		return nil, fmt.Errorf("unsupported event format")
	}

	if err != nil {
		return nil, err
	}

	event.SetFormat(format)

	return event, nil
}

func cloudeventV1FormatNewFromJSON(j []byte) (Event, error) {
	event := cloudevents.NewEvent("1.0")
	err := event.UnmarshalJSON(j)
	if err != nil {
		return nil, err
	}

	data := event.Data()

	return NewFromJSON(data)
}

// ToCloudEventV1 converts an event to a CloudEvent version 1
func ToCloudEventV1(e Event) cloudevents.Event {
	event := cloudevents.NewEvent("1.0")

	event.SetType(e.Protocol())
	event.SetSource("io.choria.lifecycle")
	event.SetSubject(e.Identity())
	event.SetID(e.ID())
	event.SetTime(e.TimeStamp())
	event.SetData(cloudevents.ApplicationJSON, e)

	return event
}

// PublishEvent publishes an event
func PublishEvent(e Event, conn inter.RawNATSConnector) error {
	var j []byte
	var err error

	switch e.Format() {
	case CloudEventV1Format:
		j, err = ToCloudEventV1(e).MarshalJSON()
	default:
		err = fmt.Errorf("do not know how to publish this format event")
	}
	if err != nil {
		return err
	}

	target, err := e.Target()
	if err != nil {
		return err
	}

	err = conn.PublishRaw(target, j)
	if err != nil {
		return err
	}

	return nil
}
