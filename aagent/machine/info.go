// Copyright (c) 2019-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machine

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// WatcherState is the status of a given watcher, boolean result is false for unknown watchers
func (m *Machine) WatcherState(watcher string) (any, bool) {
	return m.manager.WatcherState(watcher)
}

// InstanceID is a unique id for the instance of a machine
func (m *Machine) InstanceID() string {
	return m.instanceID
}

// Directory returns the directory where the machine definition is, "" when unknown
func (m *Machine) Directory() string {
	return m.directory
}

// StartTime is the time the machine started in UTC
func (m *Machine) StartTime() time.Time {
	return m.startTime
}

// Identity implements InfoSource
func (m *Machine) Identity() string {
	if m.identity == "" {
		return "unknown"
	}

	return m.identity
}

// Version implements InfoSource
func (m *Machine) Version() string {
	return m.MachineVersion
}

// Name implements InfoSource
func (m *Machine) Name() string {
	return m.MachineName
}

// State implements InfoSource
func (m *Machine) State() string {
	return m.fsm.Current()
}

// AvailableTransitions reports the transitions thats possible in the current state
func (m *Machine) AvailableTransitions() []string {
	return m.fsm.AvailableTransitions()
}

// TimeStamp returns a UTC time
func (m *Machine) TimeStamp() time.Time {
	return time.Now().UTC()
}

// TimeStampSeconds returns the current time in unix seconds
func (m *Machine) TimeStampSeconds() int64 {
	return m.TimeStamp().Unix()
}

// UniqueID creates a new unique ID, usually a v4 uuid, if that fails a random string based ID is made
func (m *Machine) UniqueID() (id string) {
	uuid, err := uuid.NewV4()
	if err == nil {
		return uuid.String()
	}

	parts := []string{}
	parts = append(parts, randStringRunes(8))
	parts = append(parts, randStringRunes(4))
	parts = append(parts, randStringRunes(4))
	parts = append(parts, randStringRunes(12))

	return strings.Join(parts, "-")
}

// Hash computes a md5 hash of the manifest
func (m *Machine) Hash() (string, error) {
	return filemd5(m.manifest)
}

func randStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.N(len(letterRunes))]
	}
	return string(b)
}

func filemd5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open data for md5 hash: %s", err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("could not copy data to md5: %s", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
