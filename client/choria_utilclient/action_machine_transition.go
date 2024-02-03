// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_util' Version 0.28.0 generated using Choria version 0.28.0

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

// MachineTransitionRequester performs a RPC request to choria_util#machine_transition
type MachineTransitionRequester struct {
	r    *requester
	outc chan *MachineTransitionOutput
}

// MachineTransitionOutput is the output from the machine_transition action
type MachineTransitionOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// MachineTransitionResult is the result from a machine_transition action
type MachineTransitionResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*MachineTransitionOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *MachineTransitionResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *MachineTransitionResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *MachineTransitionOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *MachineTransitionOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *MachineTransitionOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseMachineTransitionOutput parses the result value from the MachineTransition action into target
func (d *MachineTransitionOutput) ParseMachineTransitionOutput(target any) error {
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
func (d *MachineTransitionRequester) Do(ctx context.Context) (*MachineTransitionResult, error) {
	dres := &MachineTransitionResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &MachineTransitionOutput{
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
func (d *MachineTransitionResult) AllOutputs() []*MachineTransitionOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *MachineTransitionResult) EachOutput(h func(r *MachineTransitionOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Instance is an optional input to the machine_transition action
//
// Description: Machine Instance ID
func (d *MachineTransitionRequester) Instance(v string) *MachineTransitionRequester {
	d.r.args["instance"] = v

	return d
}

// Name is an optional input to the machine_transition action
//
// Description: Machine Name
func (d *MachineTransitionRequester) Name(v string) *MachineTransitionRequester {
	d.r.args["name"] = v

	return d
}

// Path is an optional input to the machine_transition action
//
// Description: Machine Path
func (d *MachineTransitionRequester) Path(v string) *MachineTransitionRequester {
	d.r.args["path"] = v

	return d
}

// Version is an optional input to the machine_transition action
//
// Description: Machine Version
func (d *MachineTransitionRequester) Version(v string) *MachineTransitionRequester {
	d.r.args["version"] = v

	return d
}

// Success is the value of the success output
//
// Description: Indicates if the transition was successfully accepted
func (d *MachineTransitionOutput) Success() bool {
	val := d.reply["success"]

	return val.(bool)

}
