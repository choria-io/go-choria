// generated code; DO NOT EDIT"
//
// Client for Choria RPC Agent 'choria_util' Version 0.23.0 generated using Choria version 0.23.0

package choria_utilclient

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

// MachineStatesRequester performs a RPC request to choria_util#machine_states
type MachineStatesRequester struct {
	r    *requester
	outc chan *MachineStatesOutput
}

// MachineStatesOutput is the output from the machine_states action
type MachineStatesOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// MachineStatesResult is the result from a machine_states action
type MachineStatesResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*MachineStatesOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *MachineStatesResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *MachineStatesResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *MachineStatesOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *MachineStatesOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *MachineStatesOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseMachineStatesOutput parses the result value from the MachineStates action into target
func (d *MachineStatesOutput) ParseMachineStatesOutput(target interface{}) error {
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
func (d *MachineStatesRequester) Do(ctx context.Context) (*MachineStatesResult, error) {
	dres := &MachineStatesResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &MachineStatesOutput{
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
func (d *MachineStatesResult) EachOutput(h func(r *MachineStatesOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// MachineIds is the value of the machine_ids output
//
// Description: List of running machine IDs
func (d *MachineStatesOutput) MachineIds() []interface{} {
	val := d.reply["machine_ids"]

	return val.([]interface{})

}

// MachineNames is the value of the machine_names output
//
// Description: List of running machine names
func (d *MachineStatesOutput) MachineNames() []interface{} {
	val := d.reply["machine_names"]

	return val.([]interface{})

}

// States is the value of the states output
//
// Description: Hash map of machine statusses indexed by machine ID
func (d *MachineStatesOutput) States() map[string]interface{} {
	val := d.reply["states"]

	return val.(map[string]interface{})

}
