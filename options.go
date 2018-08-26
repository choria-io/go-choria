package lifecycle

import (
	"errors"
)

// Option configures events
type Option func(e interface{}) error

// IdentityEvent is an event that has an identity
type IdentityEvent interface {
	SetIdentity(string)
}

// VersionEvent is an event that has a version
type VersionEvent interface {
	SetVersion(string)
}

// ComponentEvent is an event that a component
type ComponentEvent interface {
	SetComponent(string)
}

// Component set the component for events
func Component(component string) Option {
	return func(e interface{}) error {
		event, ok := e.(ComponentEvent)
		if !ok {
			return errors.New("cannot set component, event does not implement ComponentEvent")
		}

		event.SetComponent(component)

		return nil
	}
}

// Version set the version for events
func Version(version string) Option {
	return func(e interface{}) error {
		event, ok := e.(VersionEvent)
		if !ok {
			return errors.New("cannot set component, event does not implement VersionEvent")
		}

		event.SetVersion(version)

		return nil
	}
}

// Identity sets the identity for events
func Identity(identity string) Option {
	return func(e interface{}) error {
		event, ok := e.(IdentityEvent)
		if !ok {
			return errors.New("cannot set component, event does not implement IdentityEvent")
		}

		event.SetIdentity(identity)

		return nil
	}
}
