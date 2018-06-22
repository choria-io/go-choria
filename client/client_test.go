package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/choria-io/go-protocol/protocol/v1"

	"github.com/choria-io/go-choria/mcorpc"

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
			Expect(stats.discoveredNodes).To(Equal([]string{"test.sender.0", "test.sender.1"}))
			Expect(stats.unexpectedRespones.Hosts()).To(Equal([]string{}))
			Expect(stats.OKCount()).To(Equal(2))
			Expect(stats.All()).To(BeTrue())
		})

		It("Should support reusing options", func() {
			Expect(rpc.opts).To(BeNil())
			cl.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			res1, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
				Targets(strings.Fields("test.sender.0 test.sender.1")),
			)
			Expect(err).ToNot(HaveOccurred())
			o := rpc.opts

			res2, err := rpc.Do(
				ctx,
				"test_action",
				request{Testing: true},
			)

			Expect(o).To(Equal(rpc.opts))
			Expect(res1.Stats()).ToNot(Equal(res2.Stats()))
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
	})
})
