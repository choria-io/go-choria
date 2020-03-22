package backoff

// https://blog.gopheracademy.com/advent-2014/backoff/

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// Policy implements a backoff policy, randomizing its delays
// and saturating at the final value in Millis.
type Policy struct {
	Millis []int
}

// FiveSec is a backoff policy ranging up to 5 seconds.
var FiveSec = Policy{
	Millis: []int{500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
}

// TwentySec is a backoff policy ranging up to 20 seconds
var TwentySec = Policy{
	Millis: []int{
		500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000,
		5500, 5750, 6000, 6500, 7000, 7500, 8000, 8500, 9000, 9500, 10000,
		10500, 10750, 11000, 11500, 12000, 12500, 13000, 13500, 14000, 14500, 15000,
		15500, 15750, 16000, 16500, 17000, 17500, 18000, 18500, 19000, 19500, 20000,
	},
}

// Duration returns the time duration of the n'th wait cycle in a
// backoff policy. This is b.Millis[n], randomized to avoid thundering
// herds.
func (b Policy) Duration(n int) time.Duration {
	if n >= len(b.Millis) {
		n = len(b.Millis) - 1
	}

	return time.Duration(jitter(b.Millis[n])) * time.Millisecond
}

// Sleep sleeps for the duration of the n'th wait cycle
// in a way that can be interrupted by the context.  An error is returned
// if the context cancels the sleep
func (b Policy) Sleep(ctx context.Context, n int) error {
	timer := time.NewTimer(b.Duration(n))

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return errors.New("sleep interrupted by context")
	}
}

// For is a for{} loop that stops on context and has a backoff based sleep between loops
// if the cb returns an error the loop ends returning error
func (b Policy) For(ctx context.Context, cb func() error) error {
	tries := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		tries++

		err := cb()
		if err == nil {
			return nil
		}

		b.Sleep(ctx, tries)
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
