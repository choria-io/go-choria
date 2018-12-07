package lifecycle

import "fmt"

type basicEvent struct {
	Protocol  string `json:"protocol"`
	Ident     string `json:"identity"`
	Comp      string `json:"component"`
	Timestamp int64  `json:"timestamp"`

	etype string
	dtype Type
}

// Component is the component that produced the event
func (e *basicEvent) Component() string {
	return e.Comp
}

// SetComponent sets the component for the event
func (e *basicEvent) SetComponent(c string) {
	e.Comp = c
}

// SetIdentity sets the identity for the event
func (e *basicEvent) SetIdentity(i string) {
	e.Ident = i
}

// Identity sets the identity for the event
func (e *basicEvent) Identity() string {
	return e.Ident
}

// Target is where to publish the event to
func (e *basicEvent) Target() (string, error) {
	if e.Comp == "" {
		return "", fmt.Errorf("event is not complete, component has not been set")
	}

	return fmt.Sprintf("choria.lifecycle.event.%s.%s", e.etype, e.Comp), nil
}

// String is text suitable to display on the console etc
func (e *basicEvent) String() string {
	return fmt.Sprintf("[%s] %s: %s", e.etype, e.Ident, e.Component())
}

// Type is the type of event
func (e *basicEvent) Type() Type {
	return e.dtype
}

// TypeString the string representation of the event type
func (e *basicEvent) TypeString() string {
	return e.etype
}
