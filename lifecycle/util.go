package lifecycle

import (
	"math/rand"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var mockTime int64
var mockID string
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func timeStamp() int64 {
	if mockTime != 0 {
		return mockTime
	}

	return time.Now().UTC().Unix()
}

// generates an event ID, typically a UUID v4 but if that fails
// a string of random characters. We can't really fail to produce
// an event at this point so have to fall back to the random
// generated string when UUID fails - typically a highly exceptional
// case
func eventID() string {
	if mockID != "" {
		return mockID
	}

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
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
