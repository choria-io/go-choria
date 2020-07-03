package updatenotifier

import (
	"context"
	"sync"
	"time"
)

type Notifier struct {
	subscribers []func()
	should      bool
	sync.Mutex
}

// Notify schedules a update to subscribers in the next cycle
func (n *Notifier) Notify() {
	n.Lock()
	n.should = true
	n.Unlock()
}

// Subscribe adds a subscriber to the list of interested parties
func (n *Notifier) Subscribe(s func()) {
	n.Lock()
	n.subscribers = append(n.subscribers, s)
	n.Unlock()
}

// Update starts the process of notifying subscribers
func (n *Notifier) Update(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// batch up letting subscribers know we been updated in small time windows
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			n.Lock()
			if !n.should {
				n.Unlock()
				continue
			}
			subs := n.subscribers
			n.should = false
			n.Unlock()

			for _, s := range subs {
				s()
			}

		case <-ctx.Done():
			return
		}
	}
}
