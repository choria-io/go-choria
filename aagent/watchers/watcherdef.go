package watchers

import (
	"time"

	"github.com/pkg/errors"
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

	announceDuration time.Duration
}

// ParseAnnounceInterval parses the announce interval and ensures its not too small
func (w *WatcherDef) ParseAnnounceInterval() (err error) {
	if w.AnnounceInterval != "" {
		w.announceDuration, err = time.ParseDuration(w.AnnounceInterval)
		if err != nil {
			return errors.Wrapf(err, "unknown announce interval for watcher %s", w.Name)
		}

		if w.announceDuration < time.Minute {
			return errors.Errorf("announce interval %v is too small for watcher %s", w.announceDuration, w.Name)
		}
	}

	return nil
}
