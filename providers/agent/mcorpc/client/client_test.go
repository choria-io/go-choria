package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"github.com/choria-io/go-choria/providers/security/filesec"

	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/server/agents"

	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMcoRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC/Client")
}

var _ = Describe("Providers/Agent/McoRPC/Client", func() {
	var (
		fw      *imock.MockFramework
		cfg     *config.Config
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

		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID(), imock.WithDDLFiles("agent", "package", "testdata/mcollective/agent/package.json"))
		fw.EXPECT().NewMessage(gomock.Any(), gomock.Eq("package"), gomock.Eq("ginkgo"), gomock.Eq(inter.RequestMessageType), gomock.Eq(nil)).DoAndReturn(func(payload string, agent string, collective string, msgType string, request inter.Message) (msg inter.Message, err error) {
			return message.NewMessage(payload, agent, collective, msgType, request, fw)
		}).AnyTimes()

		fw.Configuration().LibDir = []string{"testdata"}

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
			Expect(rpc.ResolveDDL(context.Background())).ToNot(HaveOccurred())
			rpc.setOptions()
			Expect(rpc.opts.BatchSize).To(Equal(0))
			rpc.setOptions(InBatches(10, 1))
			Expect(rpc.opts.BatchSize).To(Equal(10))
		})
	})

	Describe("RPCReply", func() {
		It("Should match against replies", func() {
			r := RPCReply{
				Statuscode: 0,
				Statusmsg:  "OK",
				Data:       json.RawMessage(`{"hello":"world", "ints": [1,2,3], "strings": ["1","2","3"], "bool":true, "fbool":false}`),
			}

			check := func(f string) (bool, error) {
				res, _, err := r.MatchExpr(f, nil)
				return res, err
			}

			Expect(check("ok() && code == 0 && msg == 'OK' && data('hello') in ['world', 'bob']")).To(BeTrue())
			Expect(check("!ok() && data('hello') == 'world'")).To(BeFalse())
			Expect(check("ok() && data('hello') == 'other'")).To(BeFalse())
			Expect(check("ok() && include(data('strings'), '1')")).To(BeTrue())
			Expect(check("ok() && include(data('strings'), '5')")).To(BeFalse())
			Expect(check("ok() && include(data('ints'), 1)")).To(BeTrue())
			Expect(check("include(data('ints'), 1)")).To(BeTrue())
			Expect(check("include(data('ints'), 5)")).To(BeFalse())
			Expect(check("data('bool')")).To(BeTrue())
			Expect(check("!data('bool')")).To(BeFalse())
			Expect(check("data('fbool')")).To(BeFalse())

			res, _, err := r.MatchExpr("ok() && data('hello')", nil)
			Expect(err).To(MatchError("match expressions should return boolean"))
			Expect(res).To(BeFalse())
		})
	})

	Describe("New", func() {
		It("Should accept DDLs as an argument", func() {
			ddl := &agent.DDL{
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

			rpc, err = New(fw, "backplane", DDL(ddl))
			Expect(err).ToNot(HaveOccurred())
			Expect(rpc).ToNot(BeNil())
		})
	})

	Describe("Do", func() {
		It("Should only accept DDLs for the requested agent", func() {
			ddl := &agent.DDL{
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

			rpc, err = New(fw, "package", DDL(ddl))
			_, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				Targets(strings.Fields("host1 host2")),
				ReplyTo("custom.reply.to"),
				InBatches(1, -1),
			)
			Expect(err).To(MatchError("the DDL does not describe the package agent"))
		})

		It("Should perform the request", func() {
			reqid := ""
			handled := 0

			sec, err := filesec.New(filesec.WithChoriaConfig(&build.Info{}, cfg), filesec.WithLog(fw.Logger("")))
			Expect(err).ToNot(HaveOccurred())

			fw.EXPECT().NewSecureRequestFromTransport(gomock.Any(), gomock.Any()).DoAndReturn(func(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error) {
				return v1.NewSecureRequestFromTransport(message, sec, skipvalidate)
			}).AnyTimes()
			fw.EXPECT().NewRequestFromSecureRequest(gomock.Any()).DoAndReturn(func(sr protocol.SecureRequest) (request protocol.Request, err error) {
				return v1.NewRequestFromSecureRequest(sr)
			}).AnyTimes()
			fw.EXPECT().NewSecureReply(gomock.Any()).DoAndReturn(func(reply protocol.Reply) (secure protocol.SecureReply, err error) {
				return v1.NewSecureReply(reply, sec)
			}).AnyTimes()
			fw.EXPECT().NewTransportForSecureReply(gomock.Any()).DoAndReturn(func(reply protocol.SecureReply) (message protocol.TransportMessage, err error) {
				t, err := v1.NewTransportMessage(cfg.Identity)
				Expect(err).ToNot(HaveOccurred())
				t.SetReplyData(reply)
				return t, nil
			}).AnyTimes()
			fw.EXPECT().NewReplyFromTransportJSON(gomock.Any(), gomock.Any()).DoAndReturn(func(payload []byte, skipvalidate bool) (msg protocol.Reply, err error) {
				t, err := v1.NewTransportFromJSON(string(payload))
				Expect(err).ToNot(HaveOccurred())
				sreply, err := v1.NewSecureReplyFromTransport(t, sec, skipvalidate)
				Expect(err).ToNot(HaveOccurred())
				return v1.NewReplyFromSecureReply(sreply)
			}).AnyTimes()
			fw.EXPECT().NewRequestTransportForMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(msg inter.Message, version string) (protocol.TransportMessage, error) {
				req, err := v1.NewRequest(msg.Agent(), msg.SenderID(), msg.CallerID(), msg.TTL(), msg.RequestID(), msg.Collective())
				Expect(err).ToNot(HaveOccurred())
				req.SetMessage(msg.Payload())

				sreq, err := v1.NewSecureRequest(req, sec)
				Expect(err).ToNot(HaveOccurred())

				sm, err := v1.NewTransportMessage(fw.Configuration().Identity)
				Expect(err).ToNot(HaveOccurred())
				err = sm.SetRequestData(sreq)
				Expect(err).ToNot(HaveOccurred())

				return sm, nil
			}).AnyTimes()

			handler := func(r protocol.Reply, rpcr *RPCReply) {
				res := reply{}
				err := json.Unmarshal(rpcr.Data, &res)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Received).To(BeTrue())
				handled++
			}

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.Collective()).To(Equal("ginkgo"))
				Expect(msg.Payload()).To(Equal("{\"agent\":\"package\",\"action\":\"test_action\",\"data\":{\"testing\":true}}"))

				reqid = msg.RequestID()

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

					cm := imock.NewMockConnectorMessage(mockctl)
					cm.EXPECT().Data().Return([]byte(tj))
					rpchandler(ctx, cm)
				}
			})

			decbcalled := false
			dediscovered := 0
			delimited := 0

			result, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				ReplyHandler(handler),
				Targets(strings.Fields("test.sender.0 test.sender.1")),
				DiscoveryEndCB(func(d, l int) error {
					dediscovered = d
					delimited = l
					decbcalled = true
					return nil
				}),
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(decbcalled).To(BeTrue())
			Expect(dediscovered).To(Equal(2))
			Expect(delimited).To(Equal(2))

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
			Expect(stats.Action()).To(Equal("test_action"))
			Expect(stats.Agent()).To(Equal("package"))
		})

		It("Should support discovery callbacks and limits", func() {
			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host1"}))
			})

			discoveredCnt := 0
			limitedCnt := 0

			_, err := rpc.Do(ctx, "test_action", request{Testing: true},
				Targets([]string{"host1", "host2", "host3", "host4"}),
				LimitSize("1"),
				LimitMethod("first"),
				DiscoveryEndCB(func(d, l int) error {
					discoveredCnt = d
					limitedCnt = l
					return nil
				}),
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(discoveredCnt).To(Equal(4))
			Expect(limitedCnt).To(Equal(1))
		})

		It("Should interruptable by the discovery callback", func() {
			_, err := rpc.Do(ctx, "test_action", request{Testing: true},
				Targets([]string{"host1", "host2", "host3", "host4"}),
				LimitSize("1"),
				LimitMethod("first"),
				DiscoveryEndCB(func(d, l int) error {
					return fmt.Errorf("simulated")
				}),
			)

			Expect(err).To(MatchError("simulated"))
		})

		It("Should support batched mode", func() {
			batch1 := cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host1", "host2"}))
			})

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).After(batch1).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host3", "host4"}))
			})

			rpc.Do(ctx, "test_action", request{Testing: true}, Targets([]string{"host1", "host2", "host3", "host4"}), InBatches(2, -1))
		})

		It("Should support making requests without processing replies unbatched", func() {
			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host1", "host2"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			_, err := rpc.Do(ctx, "test_action", request{Testing: true}, Targets(strings.Fields("host1 host2")), ReplyTo("custom.reply.to"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support making requests without processing replies batched", func() {
			batch1 := cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host1"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).After(batch1).Do(func(ctx context.Context, msg inter.Message, handler client.Handler) {
				Expect(msg.DiscoveredHosts()).To(Equal([]string{"host2"}))
				Expect(msg.ReplyTo()).To(Equal("custom.reply.to"))
				Expect(handler).To(BeNil())
			})

			_, err := rpc.Do(ctx, "test_action", request{Testing: true}, Targets(strings.Fields("host1 host2")), ReplyTo("custom.reply.to"), InBatches(1, -1))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
