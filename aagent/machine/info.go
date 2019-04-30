package machine

import (
	"math/rand"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

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

// TimeStamp returns a UTC time
func (m *Machine) TimeStamp() time.Time {
	return time.Now().UTC()
}

// TimeStampSeconds returns the current time in unix seconds
func (m *Machine) TimeStampSeconds() int64 {
	return m.TimeStamp().UnixNano()
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

func randStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
