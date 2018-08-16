package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/choria-io/go-protocol/protocol/v1"

	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"

	"github.com/choria-io/go-choria/choria"
	client "github.com/choria-io/go-client/client"
	"github.com/choria-io/go-protocol/protocol"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMcoRPC(t *testing.T) {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/Client")
}

var _ = Describe("McoRPC/Client", func() {
	var (
		fw      *choria.Framework
		rpc     *RPC
		mockctl *gomock.Controller
		cl      *MockChoriaClient
		ctx     context.Context
		cancel  func()
		err     error
	)

	type request struct {
		Testing bool `json:"testing"`
	}

	type reply struct {
		Received bool `json:"received"`
	}

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		cl = NewMockChoriaClient(mockctl)

		fw, _ = choria.New("testdata/default.cfg")
		protocol.Secure = "false"
		rpc, err = New(fw, "package")
		Expect(err).ToNot(HaveOccurred())

		rpc.cl = cl

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		mockctl.Finish()
	})

	Describe("SetOptions", func() {
		It("Should set the options", func() {
			rpc.setOptions()
			Expect(rpc.opts.BatchSize).To(Equal(0))
			rpc.setOptions(InBatches(10, 1))
			Expect(rpc.opts.BatchSize).To(Equal(10))
		})
	})

	Describe("New", func() {
		var ddl *agent.DDL

		BeforeEach(func() {
			ddl = &agent.DDL{
				Metadata: &agents.Metadata{
					Name:        "backplane",
					Description: "Choria Management Backplane",
					Author:      "R.I.Pienaar <rip@devco.net>",
					Version:     "1.0.0",
					License:     "Apache-2.0",
					URL:         "https://choria.io",
					Timeout:     10,
				},
				Actions: []*agent.Action{},
				Schema:  "https://choria.io/schemas/mcorpc/ddl/v1/agent.json",
			}
		})

		It("Should accept DDLs as an argument", func() {
			rpc, err = New(fw, "backplane", DDL(ddl))
			Expect(err).ToNot(HaveOccurred())
			Expect(rpc).ToNot(BeNil())
		})

		It("Should only accept DDLs for the requested agent", func() {
			rpc, err = New(fw, "package", DDL(ddl))
			Expect(err).To(MatchError("the DDL does not describe the package agent"))
			Expect(rpc).To(BeNil())
		})
	})

	Describe("Do", func() {
		It("Should perform the request", func() {
			reqid := ""
			handled := 0

			handler := func(r protocol.Reply, rpcr *RPCReply) {
				res := reply{}
				err := json.Unmarshal(rpcr.Data, &res)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Received).To(BeTrue())
				handled++
			}

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.Collective()).To(Equal("mcollective"))
				Expect(msg.Payload).To(Equal("{\"agent\":\"package\",\"action\":\"test_action\",\"data\":{\"testing\":true}}"))

				reqid = msg.RequestID

				rpcreply := RPCReply{
					Statusmsg:  "OK",
					Statuscode: mcorpc.OK,
					Data:       json.RawMessage("{\"received\":true}"),
				}

				j, err := json.Marshal(rpcreply)
				Expect(err).ToNot(HaveOccurred())

				mt, err := msg.Transport()
				Expect(err).ToNot(HaveOccurred())

				sreq, err := fw.NewSecureRequestFromTransport(mt, true)
				Expect(err).ToNot(HaveOccurred())

				req, err := fw.NewRequestFromSecureRequest(sreq)
				Expect(err).ToNot(HaveOccurred())

				rpchandler := rpc.handlerFactory(ctx, cancel)

				for i := 0; i < 2; i++ {
					reply, err := v1.NewReply(req, fmt.Sprintf("test.sender.%d", i))
					Expect(err).ToNot(HaveOccurred())
					reply.SetMessage(string(j))

					srep, err := fw.NewSecureReply(reply)
					Expect(err).ToNot(HaveOccurred())

					transport, err := fw.NewTransportForSecureReply(srep)
					Expect(err).ToNot(HaveOccurred())

					tj, err := transport.JSON()
					Expect(err).ToNot(HaveOccurred())

					rpchandler(ctx, &choria.ConnectorMessage{Data: []byte(tj), Reply: "x", Subject: "x"})
				}
			})

			result, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				ReplyHandler(handler),
				Targets(strings.Fields("test.sender.0 test.sender.1")),
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(handled).To(Equal(2))
			stats := result.Stats()
			Expect(stats.RequestID).To(Equal(reqid))
			Expect(stats.discoveredNodes).To(Equal(strings.Fields("test.sender.0 test.sender.1")))
			Expect(*stats.DiscoveredNodes()).To(Equal(strings.Fields("test.sender.0 test.sender.1")))
			Expect(stats.unexpectedRespones.Hosts()).To(Equal([]string{}))
			Expect(stats.OKCount()).To(Equal(2))
			Expect(stats.All()).To(BeTrue())

			d, err := stats.RequestDuration()
			Expect(err).ToNot(HaveOccurred())
			Expect(d).ToNot(BeZero())
		})

		It("Should support batched mode", func() {
			batch1 := cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts).To(Equal([]string{"host1", "host2"}))
			})

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).After(batch1).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts).To(Equal([]string{"host3", "host4"}))
			})

			rpc.Do(ctx, "test_action", request{Testing: true}, Targets([]string{"host1", "host2", "host3", "host4"}), InBatches(2, -1))
		})

		It("Should support making requests without processing replies unbatched", func() {
			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts).To(Equal([]string{"host1", "host2"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			_, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				Targets(strings.Fields("host1 host2")),
				ReplyTo("custom.reply.to"),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support making requests without processing replies batched", func() {
			batch1 := cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts).To(Equal([]string{"host1"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).After(batch1).Do(func(ctx context.Context, msg *choria.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts).To(Equal([]string{"host2"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			_, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				Targets(strings.Fields("host1 host2")),
				ReplyTo("custom.reply.to"),
				InBatches(1, -1),
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
