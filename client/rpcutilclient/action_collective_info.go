// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'rpcutil' Version 0.29.4 generated using Choria version 0.29.4

package rpcutilclient

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

// CollectiveInfoRequester performs a RPC request to rpcutil#collective_info
type CollectiveInfoRequester struct {
	r    *requester
	outc chan *CollectiveInfoOutput
}

// CollectiveInfoOutput is the output from the collective_info action
type CollectiveInfoOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// CollectiveInfoResult is the result from a collective_info action
type CollectiveInfoResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*CollectiveInfoOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *CollectiveInfoResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *CollectiveInfoResult) Stats() Stats {
	return d.stats
}

// RPCClientStats is the rpc request stats
func (d *CollectiveInfoResult) RPCClientStats() *rpcclient.Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *CollectiveInfoOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *CollectiveInfoOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *CollectiveInfoOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseCollectiveInfoOutput parses the result value from the CollectiveInfo action into target
func (d *CollectiveInfoOutput) ParseCollectiveInfoOutput(target any) error {
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
func (d *CollectiveInfoRequester) Do(ctx context.Context) (*CollectiveInfoResult, error) {
	dres := &CollectiveInfoResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &CollectiveInfoOutput{
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
func (d *CollectiveInfoResult) AllOutputs() []*CollectiveInfoOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *CollectiveInfoResult) EachOutput(h func(r *CollectiveInfoOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Collectives is the value of the collectives output
//
// Description: All Collectives
func (d *CollectiveInfoOutput) Collectives() []any {
	val := d.reply["collectives"]

	return val.([]any)

}

// MainCollective is the value of the main_collective output
//
// Description: The main Collective
func (d *CollectiveInfoOutput) MainCollective() string {
	val := d.reply["main_collective"]

	return val.(string)

}
