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

// NewFromJSON creates an event from the event JSON
func NewFromJSON(j []byte) (Event, error) {
	protocol := gjson.GetBytes(j, "protocol")
	if !protocol.Exists() {
		return nil, fmt.Errorf("no protocol field present")
	}

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

// PublishEvent publishes an event
func PublishEvent(e Event, conn PublishConnector) error {
	j, err := json.Marshal(e)
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
