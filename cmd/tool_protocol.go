// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	"github.com/dustin/go-humanize"
	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

type tProtocolCommand struct {
	source string
	subj   bool
	json   bool
	count  int

	command
}

func (p *tProtocolCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		p.cmd = tool.Cmd().Command("protocol", "Debug Choria protocol messages").Hidden()
		p.cmd.Arg("source", "Where to get the message from, path to a file or subject").Required().StringVar(&p.source)
		p.cmd.Flag("subject", "Indicates that source is a subject to listen on and not a file").UnNegatableBoolVar(&p.subj)
		p.cmd.Flag("count", "When listening on the network exit after this many messages").Default("50").IntVar(&p.count)
		p.cmd.Flag("json", "Render JSON data").UnNegatableBoolVar(&p.json)
	}

	return nil
}

func (p *tProtocolCommand) Configure() error {
	return commonConfigure()
}

func (p *tProtocolCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	var payload []byte

	switch {
	case p.subj:
		conn, err := c.NewConnector(ctx, c.MiddlewareServers, "choria tool protocol", c.Logger("protocol"))
		if err != nil {
			return err
		}

		cnt := 0
		mu := sync.Mutex{}
		nc := conn.Nats()

		fmt.Printf(">>> Subscribing to subject %s for %s message(s)\n\n", c.Colorize("green", p.source), c.Colorize("green", strconv.Itoa(p.count)))

		sub, err := nc.Subscribe(p.source, func(msg *nats.Msg) {
			mu.Lock()
			defer mu.Unlock()

			cnt++
			fmt.Printf(">>> [%d] Message received %s from %s\n\n", cnt, c.Colorize("green", time.Now().Format(time.RFC822)), c.Colorize("green", msg.Subject))
			err = p.renderMsgBytes(msg.Data)
			if err != nil {
				fmt.Printf(">> invalid message on %s: %v\n\n", msg.Subject, c.Colorize("red", err.Error()))
			}
			fmt.Println()

			if cnt == p.count {
				cancel()
			}
		})
		if err != nil {
			return err
		}
		sub.AutoUnsubscribe(p.count)

	default:
		payload, err = os.ReadFile(p.source)
		if err != nil {
			return err
		}

		return p.renderMsgBytes(payload)
	}

	<-ctx.Done()

	return nil
}

func (p *tProtocolCommand) renderMsgBytes(msg []byte) error {
	if !gjson.GetBytes(msg, "protocol").Exists() {
		return fmt.Errorf("no protocol identifier found")
	}

	transport, err := c.NewTransportFromJSON(msg)
	if err != nil {
		return err
	}

	proto, err := p.renderTransport(transport)
	if err != nil {
		return err
	}

	switch proto {
	case protocol.SecureReplyV1:
		sreply, err := c.NewSecureReplyFromTransport(transport, true)
		if err != nil {
			return err
		}

		err = p.renderSecureReply(sreply)
		if err != nil {
			return err
		}

		reply, err := c.NewReplyFromSecureReply(sreply)
		if err != nil {
			return err
		}

		err = p.renderReply(reply)
		if err != nil {
			return err
		}

	case protocol.SecureRequestV1:
		srequest, err := c.NewSecureRequestFromTransport(transport, true)
		if err != nil {
			return err
		}

		err = p.renderSecureRequest(srequest)
		if err != nil {
			return err
		}

		request, err := c.NewRequestFromSecureRequest(srequest)
		if err != nil {
			return err
		}

		err = p.renderRequest(request)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot render %s", proto)
	}

	return nil
}

func (p *tProtocolCommand) renderSecureReply(t protocol.SecureReply) error {
	payload := t.Message()
	payloadProto := gjson.GetBytes(payload, "protocol")
	raw, err := t.JSON()
	if err != nil {
		return err
	}

	if p.json {
		fmt.Println("Secure Reply:")
		fmt.Println()
		err = iu.DumpJSONIndentedFormatted(raw, 4)
		fmt.Println()
		return err
	}

	fmt.Println("║   ╓─ Secure Reply ─────────────────────────────────────")
	fmt.Println("║   ║")
	fmt.Printf("║   ║ %s message with %s payload\n", c.Colorize("green", t.Version()), c.Colorize("green", humanize.IBytes(uint64(len(payload)))))
	fmt.Println("║   ║")
	fmt.Printf("║   ║   Payload Protocol: %s\n", payloadProto.String())
	fmt.Println("║   ║")

	return nil
}

func (p *tProtocolCommand) renderReply(t protocol.Reply) error {
	payload := t.Message()
	raw, err := t.JSON()
	if err != nil {
		return err
	}

	if p.json {
		fmt.Println("Secure Reply:")
		fmt.Println()
		err = iu.DumpJSONIndentedFormatted(raw, 4)
		fmt.Println()
		return err
	}

	fmt.Println("║   ║   ╓─ Reply ────────────────────────────────────────────")
	fmt.Println("║   ║   ║")
	fmt.Printf("║   ║   ║ %s message with %s payload\n", c.Colorize("green", t.Version()), c.Colorize("green", humanize.IBytes(uint64(len(payload)))))
	fmt.Println("║   ║   ║")
	fmt.Printf("║   ║   ║   Request: %s\n", t.RequestID())
	fmt.Printf("║   ║   ║     Agent: %s\n", t.Agent())
	fmt.Printf("║   ║   ║    Sender: %s\n", t.SenderID())
	fmt.Printf("║   ║   ║      Time: %s (%s ago)\n", t.Time().UTC().Format(time.RFC3339Nano), iu.RenderDuration(time.Since(t.Time())))
	if len(payload) > 65 {
		fmt.Printf("║   ║   ║   Payload: %s...%s\n", string(payload[:30]), string(payload[len(payload)-30:]))
	} else {
		fmt.Printf("║   ║   ║   Payload: %s\n", string(payload))
	}
	fmt.Println("║   ║   ║")

	return nil

}

func (p *tProtocolCommand) renderSecureRequest(t protocol.SecureRequest) error {
	payload := t.Message()
	payloadProto := gjson.GetBytes(payload, "protocol")
	raw, err := t.JSON()
	if err != nil {
		return err
	}

	if p.json {
		fmt.Println("Secure Request:")
		fmt.Println()
		err = iu.DumpJSONIndentedFormatted(raw, 4)
		fmt.Println()
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(gjson.GetBytes(raw, "signature").String())
	if err != nil {
		return err
	}

	fmt.Println("║   ╓─ Secure Request ─────────────────────────────────────")
	fmt.Println("║   ║")
	fmt.Printf("║   ║ %s message with %s payload\n", c.Colorize("green", t.Version()), c.Colorize("green", humanize.IBytes(uint64(len(payload)))))
	fmt.Println("║   ║")
	fmt.Printf("║   ║      Payload Protocol: %s\n", payloadProto.String())
	fmt.Printf("║   ║             Signature: %x...%x\n", sig[:15], sig[len(sig)-15:])
	fmt.Println("║   ║")

	return nil
}

func (p *tProtocolCommand) renderRequest(t protocol.Request) error {
	payload := t.Message()
	filter, _ := t.Filter()

	if p.json {
		fmt.Println("Request:")
		fmt.Println()
		j, err := t.JSON()
		if err != nil {
			return err
		}
		err = iu.DumpJSONIndentedFormatted(j, 4)
		fmt.Println()
		return err
	}

	fmt.Println("║   ║   ╓─ Request ────────────────────────────────────────────")
	fmt.Println("║   ║   ║")
	fmt.Printf("║   ║   ║ %s message with %s payload\n", c.Colorize("green", t.Version()), c.Colorize("green", humanize.IBytes(uint64(len(payload)))))
	fmt.Println("║   ║   ║")
	fmt.Printf("║   ║   ║         Request: %s\n", t.RequestID())
	fmt.Printf("║   ║   ║          Sender: %s\n", t.SenderID())
	fmt.Printf("║   ║   ║           Agent: %s\n", t.Agent())
	fmt.Printf("║   ║   ║          Caller: %s\n", t.CallerID())
	fmt.Printf("║   ║   ║      Collective: %s\n", t.Collective())
	fmt.Printf("║   ║   ║            Time: %s (%s ago)\n", t.Time().UTC().Format(time.RFC3339Nano), iu.RenderDuration(time.Since(t.Time())))
	fmt.Printf("║   ║   ║             TTL: %d\n", t.TTL())
	if !filter.Empty() {
		if len(filter.Agent) > 0 {
			fmt.Printf("║   ║   ║    Agent Filter: %s\n", strings.Join(filter.Agent, ", "))
		}
		if len(filter.Fact) > 0 {
			ff := []string{}
			for _, f := range filter.Fact {
				ff = append(ff, fmt.Sprintf("%s%s%s", f.Fact, f.Operator, f.Value))
			}
			fmt.Printf("║   ║   ║     Fact Filter: %s\n", strings.Join(ff, ", "))
		}
		if len(filter.Class) > 0 {
			fmt.Printf("║   ║   ║    Class Filter: %s\n", strings.Join(filter.Class, ", "))
		}
		if len(filter.Identity) > 0 {
			fmt.Printf("║   ║   ║ Identity Filter: %s\n", strings.Join(filter.Identity, ", "))
		}
		if len(filter.Compound) > 0 {
			fmt.Printf("║   ║   ║ Compound Filter: %+v\n", filter.Compound)
		}
	} else {
		fmt.Printf("║   ║   ║          Filter: unfiltered\n")
	}

	if len(payload) > 65 {
		fmt.Printf("║   ║   ║         Payload: %s...%s\n", string(payload[:30]), string(payload[len(payload)-30:]))
	} else {
		fmt.Printf("║   ║   ║         Payload: %s\n", string(payload))
	}

	fmt.Println("║   ║   ║")
	return nil
}

func (p *tProtocolCommand) renderTransport(t protocol.TransportMessage) (string, error) {
	payload, err := t.Message()
	if err != nil {
		return "", err
	}
	payloadProto := gjson.GetBytes(payload, "protocol")

	if p.json {
		fmt.Println("Transport:")
		fmt.Println()
		j, err := t.JSON()
		if err != nil {
			return "", err
		}
		err = iu.DumpJSONIndentedFormatted(j, 4)
		fmt.Println()

		return payloadProto.String(), err
	}

	fmt.Println("╓─ Transport ─────────────────────────────────────────")
	fmt.Println("║")
	fmt.Printf("║ %s message with %s payload from %s\n", c.Colorize("green", t.Version()), c.Colorize("green", humanize.IBytes(uint64(len(payload)))), c.Colorize("green", t.SenderID()))
	fmt.Println("║")
	fmt.Printf("║      Payload Protocol: %s\n", payloadProto.String())
	if len(t.ReplyTo()) > 0 {
		fmt.Printf("║              Reply-To: %s\n", t.ReplyTo())
	}
	if t.IsFederated() {
		reply, _ := t.FederationReplyTo()
		req, _ := t.FederationRequestID()
		targets, _ := t.FederationTargets()
		fmt.Printf("║   Federation Reply-To: %s\n", reply)
		fmt.Printf("║    Federation Request: %s\n", req)
		fmt.Printf("║    Federatoin Targets: %d\n", len(targets))
	} else {
		fmt.Printf("║             Federated: false\n")
	}

	fmt.Println("║")

	return payloadProto.String(), nil
}

func init() {
	cli.commands = append(cli.commands, &tProtocolCommand{})
}
