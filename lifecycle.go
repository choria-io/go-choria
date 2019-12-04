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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/tidwall/gjson"
)

// PublishConnector is a connection to the middleware
type PublishConnector interface {
	PublishRaw(target string, data []byte) error
}

// Type is a type of event this system supports
type Type int

const (
	_ = iota

	// Startup is an event components can publish when they start
	Startup Type = iota

	// Shutdown is an event components can publish when they shutdown
	Shutdown

	// Provisioned is an event components can publish post provisioning
	Provisioned

	// Alive is an event components can publish to indicate they are still alive
	Alive
)

// Format is the event format used for transporting events
type Format int

const (
	// UnknownFormat is a unknown format message
	UnknownFormat Format = iota

	// ChoriaFormat is classical ChoriaFormat lifecycle events in its own package
	ChoriaFormat

	// CloudEventV1Format is a classical Choria lifecycle event carried within a version 1.0 CloudEvent
	CloudEventV1Format
)

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
	protocol := gjson.GetBytes(j, "protocol")
	if protocol.Exists() && strings.HasPrefix(protocol.String(), "io.choria.lifecycle") {
		return ChoriaFormat
	}

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
	case ChoriaFormat:
		event, err = choriaFormatNewFromJSON(j)
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

	data, err := event.DataBytes()
	if err != nil {
		return nil, err
	}

	return NewFromJSON(data)
}

func choriaFormatNewFromJSON(j []byte) (Event, error) {
	protocol := gjson.GetBytes(j, "protocol")
	proto, err := protoStringToTypeString(protocol.String())
	if err != nil {
		return nil, err
	}

	etype, ok := eventTypes[proto]
	if !ok {
		return nil, fmt.Errorf("unknown protocol '%s' received", protocol.String())
	}

	factory, ok := eventJSONParsers[etype]
	if !ok {
		return nil, fmt.Errorf("cannot create %s event type from JSON", proto)
	}

	return factory(j)
}

// turns io.choria.lifecycle.v1.provisioned or choria:lifecycle:provisioned:1 into provisioned
func protoStringToTypeString(proto string) (eventType string, err error) {
	if strings.HasPrefix(proto, "choria:lifecycle") {
		parts := strings.Split(proto, ":")
		if len(parts) == 4 {
			return parts[2], nil
		}

		return "", fmt.Errorf("unknown protocol '%s' received", proto)
	}

	if strings.HasPrefix(proto, "io.choria.lifecycle") {
		parts := strings.Split(proto, ".")
		if len(parts) == 5 {
			return parts[4], nil
		}

		return "", fmt.Errorf("unknown protocol '%s' received", proto)
	}

	return "", fmt.Errorf("invalid protocol '%s' received", proto)
}

// ToCloudEventV1 converts an event to a CloudEvent version 1
func ToCloudEventV1(e Event) cloudevents.Event {
	event := cloudevents.NewEvent("1.0")

	event.SetType(e.TypeString())
	event.SetSource("io.choria.lifecycle")
	event.SetSubject(e.Component())
	event.SetID(e.ID())
	event.SetTime(e.TimeStamp())
	event.SetData(e)

	return event
}

// PublishEvent publishes an event
func PublishEvent(e Event, conn PublishConnector) error {
	var j []byte
	var err error

	switch e.Format() {
	case ChoriaFormat:
		j, err = json.Marshal(e)
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

	conn.PublishRaw(target, j)

	return nil
}
