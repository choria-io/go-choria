// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/choria-io/go-choria/config"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Directory Spool", func() {
	var (
		spool   *DirectorySpool
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		cfg     *config.Config
		td      string
	)

	BeforeEach(func() {
		td, err := os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		mockctl = gomock.NewController(GinkgoT())
		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.LogDiscard())
		cfg.Choria.SubmissionSpool = td
		cfg.Choria.SubmissionSpoolMaxSize = 50
		fw.EXPECT().SignerSeedFile().Return(filepath.Join("td/seed"), nil).AnyTimes()
		fw.EXPECT().SignerTokenFile().Return(filepath.Join("td/token"), nil).AnyTimes()
		subm, err := NewFromChoria(fw, Directory)
		Expect(err).ToNot(HaveOccurred())
		spool = subm.store.(*DirectorySpool)
	})

	AfterEach(func() {
		os.RemoveAll(td)
		mockctl.Finish()
	})

	Describe("StartSpool", func() {
		It("Should process all spools", func() {
			spool.pollInterval = 5 * time.Millisecond
			spool.bo = util.ConstantBackOffForTests

			unreliables := map[string]int64{}
			reliables := map[string]int64{}

			umsg1 := spool.NewMessage()
			umsg1.Subject = "foo.p1"
			umsg1.Payload = []byte("p1 message 1")
			umsg1.Priority = 1
			umsg1.Sender = "ginkgo"
			err := spool.Submit(umsg1)
			Expect(err).ToNot(HaveOccurred())
			unreliables[umsg1.ID] = 0

			umsg2 := spool.NewMessage()
			umsg2.Subject = "foo.p1"
			umsg2.Payload = []byte("p1 message 2")
			umsg2.Priority = 1
			umsg2.Sender = "ginkgo"
			err = spool.Submit(umsg2)
			Expect(err).ToNot(HaveOccurred())
			unreliables[umsg2.ID] = 0

			rmsg1 := spool.NewMessage()
			rmsg1.Subject = "foo.p4"
			rmsg1.Payload = []byte("p4 message")
			rmsg1.Priority = 4
			rmsg1.Reliable = true
			rmsg1.Sender = "ginkgo"
			rmsg1.MaxTries = 3
			err = spool.Submit(rmsg1)
			Expect(err).ToNot(HaveOccurred())
			reliables[rmsg1.ID] = 0

			rmsg2 := spool.NewMessage()
			rmsg2.Subject = "foo.p4"
			rmsg2.Payload = []byte("p4 message")
			rmsg2.Priority = 4
			rmsg2.Reliable = true
			rmsg2.Sender = "ginkgo"
			rmsg2.MaxTries = 10
			err = spool.Submit(rmsg2)
			Expect(err).ToNot(HaveOccurred())
			reliables[rmsg2.ID] = 0

			mu := sync.Mutex{}
			wg := &sync.WaitGroup{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			wg.Add(1)

			tries := 1
			var completed int32

			err = spool.StartPoll(ctx, wg, func(msgs []*Message) error {
				defer GinkgoRecover()

				mu.Lock()
				defer mu.Unlock()

				if len(msgs) == 0 {
					Fail("Received no messages")
				}

				log := fw.Logger("x")

				for _, msg := range msgs {
					if msg.Reliable {
						reliables[msg.ID]++
					} else {
						unreliables[msg.ID]++
					}

					priority := msg.Priority
					log.Infof("%s priority %d message (try: %d/%d, reliable: %v): %s\n", msg.ID, priority, msg.Tries, msg.MaxTries, msg.Reliable, string(msg.Payload))

					if (msg.Reliable && tries > 0 && msg.Tries%10 == 7) || !msg.Reliable {
						log.Infof("completing message %s on try %d\n", msg.ID, msg.Tries)
						spool.Complete(msg)
						atomic.AddInt32(&completed, 1)
					} else {
						spool.IncrementTries(msg)
					}
				}

				tries++

				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 300; i++ {
				if atomic.LoadInt32(&completed) == 3 {
					for k, v := range unreliables {
						if v != 1 {
							Fail(fmt.Sprintf("Unreliable message %s was tried %d times", k, v))
						}
					}

					if reliables[rmsg1.ID] != 3 {
						Fail(fmt.Sprintf("Realiable message %s was tried %d times, expected 3", rmsg1.ID, reliables[rmsg1.ID]))
					}

					if reliables[rmsg2.ID] != 8 {
						Fail(fmt.Sprintf("Realiable message %s was tried %d times, expected 8", rmsg2.ID, reliables[rmsg2.ID]))
					}

					return
				}

				time.Sleep(time.Second / 10)
			}
			Fail("meh")
		})
	})

	Describe("Submit", func() {
		It("Should validate the message", func() {
			msg := spool.NewMessage()

			err := spool.Submit(msg)
			Expect(err).To(MatchError("subject is required"))

			msg.Subject = "foo.bar"
			msg.Payload = []byte("hello world")
			msg.Sender = "ginkgo"

			msg.Headers = map[string]string{"Nats-X": "x"}
			err = spool.Submit(msg)
			Expect(err).To(MatchError(ErrReservedHeaderName))

			msg.Headers = map[string]string{"x": "y"}

			err = spool.Submit(msg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should save the message", func() {
			msg := spool.NewMessage()
			msg.Subject = "foo.bar"
			msg.Payload = []byte("hello world")
			msg.Sender = "ginkgo"
			msg.Headers = map[string]string{"x": "y"}
			err := spool.Submit(msg)
			Expect(err).ToNot(HaveOccurred())

			dir, err := os.ReadDir(filepath.Join(spool.directory, "P0"))
			Expect(err).ToNot(HaveOccurred())
			Expect(dir).To(HaveLen(1))

			msg = &Message{}
			jmsg, err := os.ReadFile(filepath.Join(spool.directory, "P0", dir[0].Name()))
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(jmsg, msg)
			Expect(err).ToNot(HaveOccurred())

			Expect(msg.Headers).To(Equal(map[string]string{"x": "y"}))
			Expect(msg.Payload).To(Equal([]byte("hello world")))
		})

		It("Should cap the priority", func() {
			for i := 0; i < 52; i++ {
				msg := spool.NewMessage()
				msg.Subject = "foo.bar"
				msg.Payload = []byte("hello world")
				msg.Sender = "ginkgo"
				err := spool.Submit(msg)

				if i < 51 {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError("spool is full"))
				}
			}
		})
	})
})
