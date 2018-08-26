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

	// Startup is an event components should publish when they start
	Startup Type = iota

	// Shutdown is an event components should publish when they shutdown
	Shutdown
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

	proto := strings.Split(protocol.String(), ":")
	if len(proto) != 4 {
		return nil, fmt.Errorf("invalid protocol '%s' received", protocol.String())
	}

	etype, ok := eventTypes[proto[2]]
	if !ok {
		return nil, fmt.Errorf("unknown protocol '%s' received", protocol.String())
	}

	factory, ok := eventJSONParsers[etype]
	if !ok {
		return nil, fmt.Errorf("cannot create %s event type from JSON", proto[2])
	}

	return factory(j)
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
