// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_registry' Version 0.28.0 generated using Choria version 0.29.0

package choria_registryclient

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

// DdlRequester performs a RPC request to choria_registry#ddl
type DdlRequester struct {
	r    *requester
	outc chan *DdlOutput
}

// DdlOutput is the output from the ddl action
type DdlOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// DdlResult is the result from a ddl action
type DdlResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*DdlOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *DdlResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *DdlResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *DdlOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *DdlOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *DdlOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseDdlOutput parses the result value from the Ddl action into target
func (d *DdlOutput) ParseDdlOutput(target any) error {
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
func (d *DdlRequester) Do(ctx context.Context) (*DdlResult, error) {
	dres := &DdlResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &DdlOutput{
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
func (d *DdlResult) AllOutputs() []*DdlOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *DdlResult) EachOutput(h func(r *DdlOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Format is an optional input to the ddl action
//
// Description: The result format the plugin should be retrieved in
func (d *DdlRequester) Format(v string) *DdlRequester {
	d.r.args["format"] = v

	return d
}

// Ddl is the value of the ddl output
//
// Description: The plugin DDL in the requested format
func (d *DdlOutput) Ddl() string {
	val := d.reply["ddl"]

	return val.(string)

}

// Name is the value of the name output
//
// Description: The name of the plugin
func (d *DdlOutput) Name() string {
	val := d.reply["name"]

	return val.(string)

}

// PluginType is the value of the plugin_type output
//
// Description: The type of plugin
func (d *DdlOutput) PluginType() string {
	val := d.reply["plugin_type"]

	return val.(string)

}

// Version is the value of the version output
//
// Description: The version of the plugin
func (d *DdlOutput) Version() string {
	val := d.reply["version"]

	return val.(string)

}
