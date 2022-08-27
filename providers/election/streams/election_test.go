// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package election

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLeader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Election/Streams")
}

var _ = Describe("Choria KV Leader Election", func() {
	var (
		srv      *server.Server
		nc       *nats.Conn
		js       nats.KeyValueManager
		kv       nats.KeyValue
		err      error
		debugger func(f string, a ...any)
	)

	BeforeEach(func() {
		skipValidate = false
		srv, nc = startJSServer(GinkgoT())
		js, err = nc.JetStream()
		Expect(err).ToNot(HaveOccurred())

		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: "LEADER_ELECTION",
			TTL:    500 * time.Millisecond,
		})
		Expect(err).ToNot(HaveOccurred())
		debugger = func(f string, a ...any) {
			fmt.Fprintf(GinkgoWriter, fmt.Sprintf("%s: %s\n", time.Now(), f), a...)
		}
	})

	AfterEach(func() {
		nc.Close()
		srv.Shutdown()
		srv.WaitForShutdown()
		if srv.StoreDir() != "" {
			os.RemoveAll(srv.StoreDir())
		}
	})

	Describe("Election", func() {
		It("Should validate the TTL", func() {
			kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket: "LE",
				TTL:    time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = NewElection("test", "test.key", kv)
			Expect(err).To(MatchError("bucket TTL should be 5 seconds or more"))

			err = js.DeleteKeyValue("LE")
			Expect(err).ToNot(HaveOccurred())

			kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket: "LE",
				TTL:    24 * time.Hour,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = NewElection("test", "test.key", kv)
			Expect(err).To(MatchError("bucket TTL should be less than or equal to 1 hour"))
		})

		It("Should allow 5 second TTLs", func() {
			kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket: "LE",
				TTL:    5 * time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = NewElection("test", "test.key", kv)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should correctly manage leadership", func() {
			var (
				wins      = 0
				lost      = 0
				active    = make(map[string]struct{})
				maxActive = 0
				other     = 0
				wg        = &sync.WaitGroup{}
				mu        = sync.Mutex{}
			)

			skipValidate = true

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			worker := func(wg *sync.WaitGroup, i int, key string) {
				defer wg.Done()

				name := fmt.Sprintf("member %d", i)

				winCb := func() {
					mu.Lock()
					wins++
					active[name] = struct{}{}
					act := len(active)
					if act > maxActive {
						maxActive = act
					}
					mu.Unlock()

					debugger("%d became leader with %d active leaders", i, act)
				}

				lostCb := func() {
					mu.Lock()
					lost++
					delete(active, name)
					mu.Unlock()
					debugger("%d lost leadership", i)
				}

				elect, err := NewElection(name, key, kv,
					OnWon(winCb),
					OnLost(lostCb),
					WithDebug(debugger))
				Expect(err).ToNot(HaveOccurred())

				err = elect.Start(ctx)
				Expect(err).ToNot(HaveOccurred())
			}

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go worker(wg, i, "election")
			}

			// make sure another election is not interrupted
			otherWorker := func(wg *sync.WaitGroup, i int) {
				defer wg.Done()

				elect, err := NewElection(fmt.Sprintf("other %d", i), "other", kv,
					OnWon(func() {
						mu.Lock()
						debugger("other %d gained leader", i)
						other++
						mu.Unlock()
					}),
					OnLost(func() {
						defer GinkgoRecover()
						debugger("other %d lost leader", i)
						Fail(fmt.Sprintf("Other %d election was lost", i))
					}))
				Expect(err).ToNot(HaveOccurred())

				err = elect.Start(ctx)
				Expect(err).ToNot(HaveOccurred())
			}
			wg.Add(2)
			go otherWorker(wg, 1)
			go otherWorker(wg, 2)

			// test failure scenarios, either the key gets deleted to allow a Create() to succeed
			// or it gets corrupted by putting a item in the key that would therefore change the sequence
			// causing next campaign by the leader to fail. The leader will stand-down, all campaigns will
			// fail until the corruption is removed by the MaxAge limit
			kills := 0
			for {
				if ctxSleep(ctx, 400*time.Millisecond) != nil {
					break
				}

				mu.Lock()
				act := len(active)
				mu.Unlock()

				// only corrupt when there is a leader
				if act == 0 {
					continue
				}

				kills++
				if kills%3 == 0 {
					debugger("deleting key")
					Expect(kv.Delete("election")).ToNot(HaveOccurred())
				} else {
					debugger("corrupting key")
					_, err := kv.Put("election", nil)
					Expect(err).ToNot(HaveOccurred())
				}
			}

			wg.Wait()

			mu.Lock()
			defer mu.Unlock()

			// check we had enough keys and wins etc to have tested all scenarios
			if kills < 4 {
				Fail(fmt.Sprintf("had %d kills", kills))
			}
			if wins < 4 {
				Fail(fmt.Sprintf("won only %d elections for %d kills", wins, kills))
			}
			if lost < 4 {
				Fail(fmt.Sprintf("lost only %d elections", lost))
			}
			if maxActive > 1 {
				Fail(fmt.Sprintf("Had %d leaders", maxActive))
			}
		})
	})
})

func startJSServer(t GinkgoTInterface) (*server.Server, *nats.Conn) {
	t.Helper()

	d, err := os.MkdirTemp("", "jstest")
	if err != nil {
		t.Fatalf("temp dir could not be made: %s", err)
	}

	opts := &server.Options{
		JetStream: true,
		StoreDir:  d,
		Port:      -1,
		Host:      "localhost",
		LogFile:   "/dev/stdout",
		Trace:     true,
	}

	s, err := server.NewServer(opts)
	if err != nil {
		t.Fatal("server start failed: ", err)
	}

	go s.Start()
	if !s.ReadyForConnections(10 * time.Second) {
		t.Error("nats server did not start")
	}

	nc, err := nats.Connect(s.ClientURL(), nats.UseOldRequestStyle())
	if err != nil {
		t.Fatalf("client start failed: %s", err)
	}

	return s, nc
}
