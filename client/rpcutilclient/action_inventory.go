// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'rpcutil' Version 0.26.2 generated using Choria version 0.26.2

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

// InventoryRequester performs a RPC request to rpcutil#inventory
type InventoryRequester struct {
	r    *requester
	outc chan *InventoryOutput
}

// InventoryOutput is the output from the inventory action
type InventoryOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// InventoryResult is the result from a inventory action
type InventoryResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*InventoryOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *InventoryResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *InventoryResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *InventoryOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *InventoryOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *InventoryOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseInventoryOutput parses the result value from the Inventory action into target
func (d *InventoryOutput) ParseInventoryOutput(target any) error {
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
func (d *InventoryRequester) Do(ctx context.Context) (*InventoryResult, error) {
	dres := &InventoryResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &InventoryOutput{
			reply: make(map[string]any),
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
func (d *InventoryResult) AllOutputs() []*InventoryOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *InventoryResult) EachOutput(h func(r *InventoryOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Agents is the value of the agents output
//
// Description: List of agent names
func (d *InventoryOutput) Agents() []any {
	val := d.reply["agents"]

	return val.([]any)

}

// Classes is the value of the classes output
//
// Description: List of classes on the system
func (d *InventoryOutput) Classes() []any {
	val := d.reply["classes"]

	return val.([]any)

}

// Collectives is the value of the collectives output
//
// Description: All Collectives
func (d *InventoryOutput) Collectives() []any {
	val := d.reply["collectives"]

	return val.([]any)

}

// DataPlugins is the value of the data_plugins output
//
// Description: List of data plugin names
func (d *InventoryOutput) DataPlugins() []any {
	val := d.reply["data_plugins"]

	return val.([]any)

}

// Facts is the value of the facts output
//
// Description: List of facts and values
func (d *InventoryOutput) Facts() map[string]any {
	val := d.reply["facts"]

	return val.(map[string]any)

}

// Machines is the value of the machines output
//
// Description: Autonomous Agents
func (d *InventoryOutput) Machines() map[string]any {
	val := d.reply["machines"]

	return val.(map[string]any)

}

// MainCollective is the value of the main_collective output
//
// Description: The main Collective
func (d *InventoryOutput) MainCollective() string {
	val := d.reply["main_collective"]

	return val.(string)

}

// Upgradable is the value of the upgradable output
//
// Description: Indicates if the server supports upgrades
func (d *InventoryOutput) Upgradable() bool {
	val := d.reply["upgradable"]

	return val.(bool)

}

// Version is the value of the version output
//
// Description: Choria Server Version
func (d *InventoryOutput) Version() string {
	val := d.reply["version"]

	return val.(string)

}
