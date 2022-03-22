// generated code; DO NOT EDIT"
//
// Client for Choria RPC Agent 'rpcutil' Version 0.25.1 generated using Choria version 0.25.1

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

// GetConfigItemRequester performs a RPC request to rpcutil#get_config_item
type GetConfigItemRequester struct {
	r    *requester
	outc chan *GetConfigItemOutput
}

// GetConfigItemOutput is the output from the get_config_item action
type GetConfigItemOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// GetConfigItemResult is the result from a get_config_item action
type GetConfigItemResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*GetConfigItemOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *GetConfigItemResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *GetConfigItemResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *GetConfigItemOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *GetConfigItemOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *GetConfigItemOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseGetConfigItemOutput parses the result value from the GetConfigItem action into target
func (d *GetConfigItemOutput) ParseGetConfigItemOutput(target interface{}) error {
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
func (d *GetConfigItemRequester) Do(ctx context.Context) (*GetConfigItemResult, error) {
	dres := &GetConfigItemResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &GetConfigItemOutput{
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
func (d *GetConfigItemResult) AllOutputs() []*GetConfigItemOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *GetConfigItemResult) EachOutput(h func(r *GetConfigItemOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Item is the value of the item output
//
// Description: The config property being retrieved
func (d *GetConfigItemOutput) Item() string {
	val := d.reply["item"]

	return val.(string)

}

// Value is the value of the value output
//
// Description: The value that is in use
func (d *GetConfigItemOutput) Value() interface{} {
	val, ok := d.reply["value"]
	if !ok || val == nil {
		// we have to avoid returning nil.(interface{})
		return nil
	}

	return val

}
