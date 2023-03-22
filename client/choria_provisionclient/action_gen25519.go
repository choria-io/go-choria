// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_provision' Version 0.27.0 generated using Choria version 0.27.0

package choria_provisionclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"
)

// Gen25519Requester performs a RPC request to choria_provision#gen25519
type Gen25519Requester struct {
	r    *requester
	outc chan *Gen25519Output
}

// Gen25519Output is the output from the gen25519 action
type Gen25519Output struct {
	details *ResultDetails
	reply   map[string]any
}

// Gen25519Result is the result from a gen25519 action
type Gen25519Result struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*Gen25519Output
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *Gen25519Result) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stats == nil {
		return fmt.Errorf("result stats is not set, result was not completed")
	}

	results := &replyfmt.RPCResults{
		Agent:   d.stats.Agent(),
		Action:  d.stats.Action(),
		Replies: d.rpcreplies,
		Stats:   d.stats,
	}

	addl, err := d.ddl.ActionInterface(d.stats.Action())
	if err != nil {
		return err
	}

	switch format {
	case JSONFormat:
		return results.RenderJSON(w, addl)
	case TableFormat:
		return results.RenderTable(w, addl)
	case TXTFooter:
		results.RenderTXTFooter(w, verbose)
		return nil
	default:
		return results.RenderTXT(w, addl, verbose, silent, replyfmt.DisplayMode(displayMode), colorize, log)
	}
}

// Stats is the rpc request stats
func (d *Gen25519Result) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *Gen25519Output) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *Gen25519Output) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *Gen25519Output) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseGen25519Output parses the result value from the Gen25519 action into target
func (d *Gen25519Output) ParseGen25519Output(target any) error {
	j, err := d.JSON()
	if err != nil {
		return fmt.Errorf("could not access payload: %s", err)
	}

	err = json.Unmarshal(j, target)
	if err != nil {
		return fmt.Errorf("could not unmarshal JSON payload: %s", err)
	}

	return nil
}

// Do performs the request
func (d *Gen25519Requester) Do(ctx context.Context) (*Gen25519Result, error) {
	dres := &Gen25519Result{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &Gen25519Output{
			reply: make(map[string]any),
			details: &ResultDetails{
				sender:  pr.SenderID(),
				code:    int(r.Statuscode),
				message: r.Statusmsg,
				ts:      pr.Time(),
			},
		}

		addl.SetOutputDefaults(output.reply)

		err := json.Unmarshal(r.Data, &output.reply)
		if err != nil {
			d.r.client.errorf("Could not decode reply from %s: %s", pr.SenderID(), err)
		}

		// caller wants a channel so we dont return a resultset too (lots of memory etc)
		// this is unused now, no support for setting a channel
		if d.outc != nil {
			d.outc <- output
			return
		}

		// else prepare our result set
		dres.mu.Lock()
		dres.outputs = append(dres.outputs, output)
		dres.rpcreplies = append(dres.rpcreplies, &replyfmt.RPCReply{
			Sender:   pr.SenderID(),
			RPCReply: r,
		})
		dres.mu.Unlock()
	}

	res, err := d.r.do(ctx, handler)
	if err != nil {
		return nil, err
	}

	dres.stats = res

	return dres, nil
}

// AllOutputs provide access to all outputs
func (d *Gen25519Result) AllOutputs() []*Gen25519Output {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *Gen25519Result) EachOutput(h func(r *Gen25519Output)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Directory is the value of the directory output
//
// Description: The directory where server.key and server.pub is written to
func (d *Gen25519Output) Directory() string {
	val := d.reply["directory"]

	return val.(string)

}

// PublicKey is the value of the public_key output
//
// Description: The ED255519 public key hex encoded
func (d *Gen25519Output) PublicKey() string {
	val := d.reply["public_key"]

	return val.(string)

}

// Signature is the value of the signature output
//
// Description: The signature of the nonce made using the new private key, hex encoded
func (d *Gen25519Output) Signature() string {
	val := d.reply["signature"]

	return val.(string)

}
