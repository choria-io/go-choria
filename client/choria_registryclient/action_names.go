// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_registry' Version 0.26.0 generated using Choria version 0.26.0

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

// NamesRequester performs a RPC request to choria_registry#names
type NamesRequester struct {
	r    *requester
	outc chan *NamesOutput
}

// NamesOutput is the output from the names action
type NamesOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// NamesResult is the result from a names action
type NamesResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*NamesOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *NamesResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *NamesResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *NamesOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *NamesOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *NamesOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseNamesOutput parses the result value from the Names action into target
func (d *NamesOutput) ParseNamesOutput(target interface{}) error {
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
func (d *NamesRequester) Do(ctx context.Context) (*NamesResult, error) {
	dres := &NamesResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &NamesOutput{
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
func (d *NamesResult) AllOutputs() []*NamesOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *NamesResult) EachOutput(h func(r *NamesOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Names is the value of the names output
//
// Description: The names of all known DDL files
func (d *NamesOutput) Names() []interface{} {
	val := d.reply["names"]

	return val.([]interface{})

}

// PluginType is the value of the plugin_type output
//
// Description: The type of plugin
func (d *NamesOutput) PluginType() string {
	val := d.reply["plugin_type"]

	return val.(string)

}
