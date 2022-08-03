// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"errors"
)

// Option configures events
type Option func(e any) error

// VersionEvent is an event that has a version
type VersionEvent interface {
	SetVersion(string)
}

// ComponentEvent is an event that has a component
type ComponentEvent interface {
	SetComponent(string)
}

// GovernedEvent is an event that relates to Governors
type GovernedEvent interface {
	SetGovernor(name string)
	SetSequence(seq uint64)
	SetEventType(stage GovernorEventType) error
}

// Component set the component for events
func Component(component string) Option {
	return func(e any) error {
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
	return func(e any) error {
		event, ok := e.(VersionEvent)
		if !ok {
			return errors.New("cannot set version, event does not implement VersionEvent")
		}

		event.SetVersion(version)

		return nil
	}
}

// Identity sets the identity for events
func Identity(identity string) Option {
	return func(e any) error {
		event, ok := e.(Event)
		if !ok {
			return errors.New("cannot set identity, event does not implement Event")
		}

		event.SetIdentity(identity)

		return nil
	}
}

// GovernorName sets the name of the Governor being interacted with
func GovernorName(name string) Option {
	return func(e any) error {
		event, ok := e.(GovernedEvent)
		if !ok {
			return errors.New("cannot set governor, event is not a Governor event")
		}

		event.SetGovernor(name)

		return nil
	}
}

// GovernorType sets the stage this event relates to
func GovernorType(stage GovernorEventType) Option {
	return func(e any) error {
		event, ok := e.(GovernedEvent)
		if !ok {
			return errors.New("cannot set governor, event is not a Governor event")
		}

		return event.SetEventType(stage)
	}
}

// GovernorSequence sets the sequence of the event when applicable
func GovernorSequence(seq uint64) Option {
	return func(e any) error {
		event, ok := e.(GovernedEvent)
		if !ok {
			return errors.New("cannot set governor, event is not a Governor event")
		}

		event.SetSequence(seq)
		return nil
	}
}
