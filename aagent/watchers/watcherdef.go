// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package watchers

import (
	"fmt"
	"time"

	"github.com/choria-io/go-choria/internal/util"
)

// WatcherDef is the core definition of a watcher, watcher type specific
// properties get stored in Properties and parsed by each watcher type
type WatcherDef struct {
	Name              string                 `json:"name" yaml:"name"`
	Type              string                 `json:"type" yaml:"type"`
	StateMatch        []string               `json:"state_match" yaml:"state_match"`
	FailTransition    string                 `json:"fail_transition" yaml:"fail_transition"`
	SuccessTransition string                 `json:"success_transition" yaml:"success_transition"`
	Interval          string                 `json:"interval" yaml:"interval"`
	AnnounceInterval  string                 `json:"announce_interval" yaml:"announce_interval"`
	Properties        map[string]interface{} `json:"properties" yaml:"properties"`
	AnnounceDuration  time.Duration          `json:"-" yaml:"-"`
}

// ParseAnnounceInterval parses the announce interval and ensures its not too small
func (w *WatcherDef) ParseAnnounceInterval() (err error) {
	if w.AnnounceInterval != "" {
		w.AnnounceDuration, err = util.ParseDuration(w.AnnounceInterval)
		if err != nil {
			return fmt.Errorf("unknown announce interval for watcher %s: %s", w.Name, err)
		}

		if w.AnnounceDuration < time.Minute {
			return fmt.Errorf("announce interval %v is too small for watcher %s", w.AnnounceDuration, w.Name)
		}
	}

	return nil
}

// ValidateStates makes sure that all the states mentioned are valid
func (w *WatcherDef) ValidateStates(valid []string) (err error) {
	hasf := func(state string) bool {
		for _, s := range valid {
			if s == state {
				return true
			}
		}

		return false
	}

	for _, s := range w.StateMatch {
		if !hasf(s) {
			return fmt.Errorf("invalid state %s in state match for watcher %s", s, w.Name)
		}
	}

	return nil
}

// ValidateTransitions checks that all stated transitions are valid
func (w *WatcherDef) ValidateTransitions(valid []string) (err error) {
	hasf := func(transition string) bool {
		for _, t := range valid {
			if t == transition {
				return true
			}
		}

		return false
	}

	if w.FailTransition != "" && !hasf(w.FailTransition) {
		return fmt.Errorf("invalid fail_transition %s specified in watcher %s", w.FailTransition, w.Name)
	}

	if w.SuccessTransition != "" && !hasf(w.SuccessTransition) {
		return fmt.Errorf("invalid success_transition %s specified in watcher %s", w.SuccessTransition, w.Name)
	}

	return nil
}
