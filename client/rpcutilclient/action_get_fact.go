// generated code; DO NOT EDIT; 2021-01-10 20:46:03.472674 +0100 CET m=+0.025672097"
//
// Client for Choria RPC Agent 'rpcutil'' Version 0.19.0 generated using Choria version 0.18.0

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

// GetFactRequester performs a RPC request to rpcutil#get_fact
type GetFactRequester struct {
	r    *requester
	outc chan *GetFactOutput
}

// GetFactOutput is the output from the get_fact action
type GetFactOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// GetFactResult is the result from a get_fact action
type GetFactResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*GetFactOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *GetFactResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *GetFactResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *GetFactOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *GetFactOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *GetFactOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseOutput parses the result value from the GetFact action into target
func (d *GetFactOutput) ParseGetFactOutput(target interface{}) error {
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
func (d *GetFactRequester) Do(ctx context.Context) (*GetFactResult, error) {
	dres := &GetFactResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &GetFactOutput{
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

// EachOutput iterates over all results received
func (d *GetFactResult) EachOutput(h func(r *GetFactOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Fact is the value of the fact output
//
// Description: The name of the fact being returned
func (d *GetFactOutput) Fact() interface{} {
	val := d.reply["fact"]
	return val.(interface{})
}

// Value is the value of the value output
//
// Description: The value of the fact
func (d *GetFactOutput) Value() interface{} {
	val := d.reply["value"]
	return val.(interface{})
}
