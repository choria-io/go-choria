package backoff

// https://blog.gopheracademy.com/advent-2014/backoff/

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// BackoffPolicy implements a backoff policy, randomizing its delays
// and saturating at the final value in Millis.
type BackoffPolicy struct {
	Millis []int
}

// FiveSec is a backoff policy ranging up to 5 seconds.
var FiveSec = BackoffPolicy{
	[]int{500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
}

// Duration returns the time duration of the n'th wait cycle in a
// backoff policy. This is b.Millis[n], randomized to avoid thundering
// herds.
func (b BackoffPolicy) Duration(n int) time.Duration {
	if n >= len(b.Millis) {
		n = len(b.Millis) - 1
	}

	return time.Duration(jitter(b.Millis[n])) * time.Millisecond
}

// InterruptableSleep sleep for the duration of the n'th wait cycle
// in a way that can be interrupted by the context.  An error is returned
// if the context cancels the sleep
func (b BackoffPolicy) InterruptableSleep(ctx context.Context, n int) error {
	timer := time.NewTimer(b.Duration(n))

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return errors.New("sleep interrupted by context")
	}
}

// jitter returns a random integer uniformly distributed in the range
// [0.5 * millis .. 1.5 * millis]
func jitter(millis int) int {
	if millis == 0 {
		return 0
	}

	return millis/2 + rand.Intn(millis)
}
