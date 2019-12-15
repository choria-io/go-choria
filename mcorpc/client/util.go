package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

// ParseReplyData parses reply data and populates a Reply and custom Data
func ParseReplyData(source []byte) (*RPCReply, error) {
	reply := &RPCReply{}

	err := json.Unmarshal(source, reply)
	if err != nil {
		return reply, fmt.Errorf("could not decode source data: %s", err)
	}

	return reply, nil
}

// InGroups calls f for sub slices of a slice where every slice
// is at most `size` big
func InGroups(set []string, size int, f func([]string) error) error {
	count := math.Ceil(float64(len(set)) / float64(size))

	for i := 0; i < int(count); i++ {
		start := i * int(size)
		end := start + int(size)

		if end > len(set) {
			end = len(set)
		}

		err := f(set[start:end])
		if err != nil {
			return err
		}
	}

	return nil
}

// InterruptableSleep sleep for the duration of the n'th wait cycle
// in a way that can be interrupted by the context.  An error is returned
// if the context cancels the sleep
func InterruptableSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return errors.New("sleep interrupted by context")
	}
}
