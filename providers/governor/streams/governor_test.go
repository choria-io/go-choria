// Copyright 2020-2022 The NATS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package governor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/jsm.go"
	natsd "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGovernor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provtarget")
}

func startJSServer() (*natsd.Server, *nats.Conn) {
	d, err := os.MkdirTemp("", "jstest")
	Expect(err).ToNot(HaveOccurred())

	opts := &natsd.Options{
		ServerName: "test.example.net",
		JetStream:  true,
		StoreDir:   d,
		Port:       -1,
		Host:       "localhost",
		// LogFile:    "/tmp/server.log",
		// Trace:        true,
		// TraceVerbose: true,
		Cluster: natsd.ClusterOpts{Name: "gotest"},
	}

	s, err := natsd.NewServer(opts)
	Expect(err).ToNot(HaveOccurred())

	go s.Start()
	if !s.ReadyForConnections(10 * time.Second) {
		Fail("nats server did not start")
	}

	nc, err := nats.Connect(s.ClientURL(), nats.UseOldRequestStyle(), nats.MaxReconnects(-1))
	Expect(err).ToNot(HaveOccurred())

	return s, nc
}

var _ = Describe("Governor", func() {
	It("Should function correctly", func() {
		srv, nc := startJSServer()
		defer srv.Shutdown()
		defer nc.Close()

		_, err := NewManager("TEST", 0, 0, 0, nc, true)
		Expect(err).To(MatchError("unknown governor"))

		limit := 100

		gmgr, err := NewManager("TEST", uint64(limit), 2*time.Minute, 0, nc, true)
		Expect(err).ToNot(HaveOccurred())

		Expect(gmgr.Name()).To(Equal("TEST"))
		Expect(gmgr.Limit()).To(Equal(int64(limit)))
		Expect(gmgr.MaxAge()).To(Equal(2 * time.Minute))
		Expect(gmgr.Stream().Name()).To(Equal("GOVERNOR_TEST"))
		Expect(gmgr.Stream().Subjects()).To(Equal([]string{"$GOVERNOR.campaign.TEST"}))

		gmgr, err = NewManager("TEST", uint64(limit), 2*time.Minute, 0, nc, true, WithSubject("$BOB"))
		Expect(err).ToNot(HaveOccurred())
		Expect(gmgr.Stream().Subjects()).To(Equal([]string{"$BOB"}))

		ts, err := gmgr.LastActive()
		Expect(err).ToNot(HaveOccurred())

		Expect(ts.IsZero()).To(BeTrue())

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		workers := 1000
		max := 0
		current := 0
		cnt := 0
		mu := sync.Mutex{}
		wg := sync.WaitGroup{}
		var errs []string

		// checks evict returns the valid name
		g := New("TEST", nc, WithInterval(10*time.Millisecond), WithSubject("$BOB"))
		fin, seq, err := g.Start(ctx, "testing.eviction")
		Expect(err).ToNot(HaveOccurred())

		name, err := gmgr.Evict(seq)
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal("testing.eviction"))

		_, err = gmgr.Evict(seq)
		Expect(jsm.IsNatsError(err, 10037)).To(BeTrue())
		fin()

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				g := New("TEST", nc, WithInterval(10*time.Millisecond), WithSubject("$BOB"))

				name := fmt.Sprintf("worker %d", i)
				finisher, _, err := g.Start(ctx, name)
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Sprintf("%d did not start: %s", i, err))
					mu.Unlock()
					return
				}

				mu.Lock()
				cnt++
				current++
				if max < current {
					max = current
				}
				mu.Unlock()

				// give the scheduler a chance
				time.Sleep(50 * time.Millisecond)

				// before finish because its very quick and another one starts before this happens if its after finished call
				mu.Lock()
				current--
				mu.Unlock()

				err = finisher()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Sprintf("%d finished failed: %s", i, err))
					mu.Unlock()
					return
				}

				err = finisher()
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Sprintf("2nd finish errored: %s", err))
					mu.Unlock()
					return
				}
			}(i)
		}

		for {
			if ctx.Err() != nil {
				Fail(fmt.Sprintf("timeout %s", ctx.Err()))
			}

			mu.Lock()
			if cnt == workers {
				if max > limit {
					Fail(fmt.Sprintf("had more than %d concurrent: %d", limit, max))
				}
				mu.Unlock()

				wg.Wait()

				ts, err := gmgr.LastActive()
				Expect(err).ToNot(HaveOccurred())
				Expect(ts).To(BeTemporally("<", time.Now(), time.Second))

				if len(errs) > 0 {
					Fail(fmt.Sprintf("Had errors in workers: %s", strings.Join(errs, ", ")))
				}

				return
			}
			mu.Unlock()

			time.Sleep(time.Millisecond)
		}
	})
})
