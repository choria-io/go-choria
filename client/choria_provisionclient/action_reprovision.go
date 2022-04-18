// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_provision' Version 0.25.1 generated using Choria version 0.25.1

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

// ReprovisionRequester performs a RPC request to choria_provision#reprovision
type ReprovisionRequester struct {
	r    *requester
	outc chan *ReprovisionOutput
}

// ReprovisionOutput is the output from the reprovision action
type ReprovisionOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// ReprovisionResult is the result from a reprovision action
type ReprovisionResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*ReprovisionOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *ReprovisionResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *ReprovisionResult) Stats() Stats {
	return d.stats
}

// RPCClientStats is the rpc request stats
func (d *ReprovisionResult) RPCClientStats() *rpcclient.Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *ReprovisionOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *ReprovisionOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ReprovisionOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseReprovisionOutput parses the result value from the Reprovision action into target
func (d *ReprovisionOutput) ParseReprovisionOutput(target interface{}) error {
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
func (d *ReprovisionRequester) Do(ctx context.Context) (*ReprovisionResult, error) {
	dres := &ReprovisionResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &ReprovisionOutput{
			reply: make(map[string]interface{}),
			details: &ResultDetails{
				sender:  pr.SenderID(),
				code:    int(r.Statuscode),
				message: r.Statusmsg,
				ts:      pr.Time(),
			},
		}

		err := json.Unmarshal(r.Data, &output.reply)
		if err != nil {
			d.r.client.errorf("Could not decode reply from %s: %s", pr.SenderID(), err)
		}

		// caller wants a channel so we dont return a resulset too (lots of memory etc)
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
func (d *ReprovisionResult) AllOutputs() []*ReprovisionOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *ReprovisionResult) EachOutput(h func(r *ReprovisionOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Message is the value of the message output
//
// Description: Status message from the Provisioner
func (d *ReprovisionOutput) Message() string {
	val := d.reply["message"]

	return val.(string)

}
