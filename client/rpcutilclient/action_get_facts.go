// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'rpcutil' Version 0.28.0 generated using Choria version 0.28.0

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

// GetFactsRequester performs a RPC request to rpcutil#get_facts
type GetFactsRequester struct {
	r    *requester
	outc chan *GetFactsOutput
}

// GetFactsOutput is the output from the get_facts action
type GetFactsOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// GetFactsResult is the result from a get_facts action
type GetFactsResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*GetFactsOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *GetFactsResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *GetFactsResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *GetFactsOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *GetFactsOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *GetFactsOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseGetFactsOutput parses the result value from the GetFacts action into target
func (d *GetFactsOutput) ParseGetFactsOutput(target any) error {
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
func (d *GetFactsRequester) Do(ctx context.Context) (*GetFactsResult, error) {
	dres := &GetFactsResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &GetFactsOutput{
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
func (d *GetFactsResult) AllOutputs() []*GetFactsOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *GetFactsResult) EachOutput(h func(r *GetFactsOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Values is the value of the values output
//
// Description: List of values of the facts
func (d *GetFactsOutput) Values() map[string]any {
	val := d.reply["values"]

	return val.(map[string]any)

}
