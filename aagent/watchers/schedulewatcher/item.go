// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package schedulewatcher

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/robfig/cron"
)

type scheduleItem struct {
	spec      string
	sched     cron.Schedule
	events    chan int
	on        bool
	duration  time.Duration
	randomize time.Duration
	machine   model.Machine
	watcher   *Watcher

	sync.Mutex
}

func newSchedItem(s string, w *Watcher) (item *scheduleItem, err error) {
	item = &scheduleItem{
		spec:      s,
		events:    w.ctrq,
		machine:   w.machine,
		watcher:   w,
		duration:  w.properties.Duration,
		randomize: w.properties.StartSplay,
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
		defer s.Unlock()

		sleep := time.Duration(0)
		if s.randomize > 0 {
			sleep = time.Duration(rand.Int63n(int64(s.randomize)))
			s.watcher.Infof("Splay sleeping %v before starting schedule", sleep)
			err := util.InterruptibleSleep(ctx, sleep)
			if err != nil {
				return
			}
		}

		s.watcher.Infof("Schedule %s starting", s.spec)
		s.on = true
		s.events <- 1

		go s.wait(ctx, s.duration-sleep)
	}
}

func (s *scheduleItem) wait(ctx context.Context, t time.Duration) {
	s.watcher.Infof("Scheduling on until %v", time.Now().Add(t))
	timer := time.NewTimer(t)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
		return
	}

	s.Lock()
	s.watcher.Infof("Schedule %s ending", s.spec)
	s.on = false
	s.events <- -1
	s.Unlock()
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
