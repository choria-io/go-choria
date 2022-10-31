// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package mcorpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	imock "github.com/choria-io/go-choria/inter/imocks"
	"github.com/choria-io/go-choria/message"
	"github.com/choria-io/go-choria/protocol"
	v1 "github.com/choria-io/go-choria/protocol/v1"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

func TestMcoRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/Agent/McoRPC")
}

var _ = Describe("McoRPC", func() {
	var (
		agent   *Agent
		mockctl *gomock.Controller
		fw      *imock.MockFramework
		cfg     *config.Config
		msg     inter.Message
		req     protocol.Request
		outbox  = make(chan *agents.AgentReply, 1)
		err     error
		ctx     context.Context
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		protocol.Secure = "false"
		build.TLS = "false"

		fw, cfg = imock.NewFrameworkForTests(mockctl, GinkgoWriter, imock.WithCallerID())
		cfg.LogLevel = "fatal"

		metadata := &agents.Metadata{Name: "test"}
		agent = New("testing", metadata, fw, fw.Logger("test"))
		ctx = context.Background()
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	It("Should have correct constants", func() {
		Expect(OK).To(Equal(StatusCode(0)))
		Expect(Aborted).To(Equal(StatusCode(1)))
		Expect(UnknownAction).To(Equal(StatusCode(2)))
		Expect(MissingData).To(Equal(StatusCode(3)))
		Expect(InvalidData).To(Equal(StatusCode(4)))
		Expect(UnknownError).To(Equal(StatusCode(5)))
	})

	Describe("RegisterAction", func() {
		It("Should fail if the action already exist", func() {
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {}
			err := agent.RegisterAction("test", action)
			Expect(err).ToNot(HaveOccurred())
			err = agent.RegisterAction("test", action)
			Expect(err).To(MatchError("cannot register action test, it already exist"))
		})
	})

	Describe("HandleMessage", func() {
		BeforeEach(func() {
			req, err = v1.NewRequest("test", "test.example.net", "choria=rip.mcollective", 60, "testrequest", "mcollective")
			Expect(err).ToNot(HaveOccurred())
			msg, err = message.NewMessageFromRequest(req, "dev.null", fw)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle bad incoming data", func() {
			agent.HandleMessage(ctx, msg, req, nil, outbox)

			reply := <-outbox
			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("Could not process request: could not parse incoming message as a MCollective SimpleRPC Request: unexpected end of JSON input"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(4)))
		})

		It("Should handle unknown actions", func() {
			msg.SetPayload([]byte(`{"agent":"test", "action":"nonexisting"}`))
			agent.HandleMessage(ctx, msg, req, nil, outbox)

			reply := <-outbox
			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("Unknown action nonexisting for agent test"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(2)))
		})

		It("Should call the action", func() {
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {
				d := make(map[string]string)
				d["test"] = "hello world"
				reply.Data = &d
			}

			agent.RegisterAction("test", action)
			msg.SetPayload([]byte(`{"agent":"test", "action":"test"}`))
			agent.HandleMessage(ctx, msg, req, nil, outbox)

			reply := <-outbox
			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("OK"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(0)))
			Expect(gjson.GetBytes(reply.Body, "data.test").String()).To(Equal("hello world"))
		})

		It("Should detect unsupported authorization systems", func() {
			cfg.RPCAuthorization = true
			msg.SetPayload([]byte(`{"agent":"test", "action":"test"}`))
			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {
				d := map[string]string{"test": "hello world"}
				reply.Data = &d
			}

			agent.RegisterAction("test", action)
			agent.HandleMessage(ctx, msg, req, nil, outbox)
			reply := <-outbox

			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("You are not authorized to call this agent or action"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(1)))
		})

		It("Should support action_policy authorization", func() {
			cfg.ConfigFile = "testdata/config.cfg"
			cfg.RPCAuthorization = true
			msg.SetPayload([]byte(`{"agent":"test", "action":"test"}`))

			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {
				d := map[string]string{"test": "hello world"}
				reply.Data = &d
			}

			agent.RegisterAction("test", action)
			agent.HandleMessage(ctx, msg, req, nil, outbox)
			reply := <-outbox

			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("You are not authorized to call this agent or action"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(1)))
		})

		It("Should support rego_policy authorization", func() {
			cfg.ConfigFile = "testdata/config.cfg"
			cfg.RPCAuthorization = true
			msg.SetPayload([]byte(`{"agent":"test", "action":"test"}`))

			action := func(ctx context.Context, req *Request, reply *Reply, agent *Agent, conn inter.ConnectorInfo) {
				d := map[string]string{"test": "hello world"}
				reply.Data = &d
			}

			agent.RegisterAction("test", action)
			agent.HandleMessage(ctx, msg, req, nil, outbox)
			reply := <-outbox

			Expect(gjson.GetBytes(reply.Body, "statusmsg").String()).To(Equal("You are not authorized to call this agent or action"))
			Expect(gjson.GetBytes(reply.Body, "statuscode").Int()).To(Equal(int64(1)))

		})
	})

	Describe("publish", func() {
		It("Should handle bad data", func() {
			reply := &Reply{
				Data: outbox,
			}

			agent.publish(reply, msg, req, outbox)
			out := <-outbox
			Expect(out.Error).To(MatchError("json: unsupported type: chan *agents.AgentReply"))
		})

		PIt("Should publish good messages")
	})

	Describe("ParseRequestData", func() {
		It("Should handle valid data correctly", func() {
			req := &Request{
				Data: json.RawMessage(`{"hello":"world"}`),
			}

			reply := &Reply{}

			var params struct {
				Hello string `json:"hello"`
			}

			ok := ParseRequestData(&params, req, reply)

			Expect(ok).To(BeTrue())
			Expect(params.Hello).To(Equal("world"))
		})

		It("Should handle invalid data correctly", func() {
			req := &Request{
				Agent:  "test",
				Action: "will_fail",
				Data:   json.RawMessage(`fail`),
			}

			reply := &Reply{}

			var params struct {
				Hello string `json:"hello"`
			}

			ok := ParseRequestData(&params, req, reply)

			Expect(ok).To(BeFalse())
			Expect(reply.Statuscode).To(Equal(InvalidData))
			Expect(reply.Statusmsg).To(Equal("Could not parse request data for test#will_fail: invalid character 'i' in literal false (expecting 'l')"))
		})

		It("Should use the validator to validate structs", func() {
			req := &Request{
				Agent:  "test",
				Action: "will_fail",
				Data:   json.RawMessage(`{"hello":"foo > bar"}`),
			}

			reply := &Reply{}

			var params struct {
				Hello string `json:"hello" validate:"shellsafe"`
			}

			ok := ParseRequestData(&params, req, reply)

			Expect(ok).To(BeFalse())
			Expect(reply.Statuscode).To(Equal(InvalidData))
			Expect(reply.Statusmsg).To(Equal("Validation failed: Hello shellsafe validation failed: may not contain '>'"))
		})
	})

	Describe("newReply", func() {
		It("Should set the correct starting code and message", func() {
			r := agent.newReply()
			Expect(r.Statuscode).To(Equal(OK))
			Expect(r.Statusmsg).To(Equal("OK"))
		})
	})
})
