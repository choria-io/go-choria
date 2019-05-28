package schedulewatcher

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron"
)

type scheduleItem struct {
	spec     string
	sched    cron.Schedule
	events   chan int
	on       bool
	duration time.Duration
	machine  Machine
	watcher  *Watcher

	sync.Mutex
}

func newSchedItem(s string, w *Watcher) (item *scheduleItem, err error) {
	item = &scheduleItem{
		spec:     s,
		events:   w.ctrq,
		machine:  w.machine,
		watcher:  w,
		duration: w.duration,
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	item.sched, err = parser.Parse(s)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (s *scheduleItem) check(ctx context.Context) {
	now := time.Now()
	next := s.sched.Next(now)

	// using unix time to round it to nearest second
	if next.Unix()-1 == now.Unix() {
		s.Lock()
		s.machine.Infof(s.watcher.name, "Schedule %s starting", s.spec)
		s.on = true
		s.events <- 1
		s.Unlock()

		go s.wait(ctx)
	}
}

func (s *scheduleItem) wait(ctx context.Context) {
	s.machine.Infof(s.watcher.name, "Scheduling on until %v", time.Now().Add(s.duration))
	timer := time.NewTimer(s.duration)

	select {
	case <-timer.C:
	case <-ctx.Done():
	}

	s.Lock()
	s.machine.Infof(s.watcher.name, "Schedule %s ending", s.spec)
	s.on = false
	s.events <- -1
	s.Unlock()

	timer.Stop()
}

func (s *scheduleItem) start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			s.check(ctx)

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
