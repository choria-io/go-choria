// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_util' Version 0.26.2 generated using Choria version 0.26.2

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

// MachineStateRequester performs a RPC request to choria_util#machine_state
type MachineStateRequester struct {
	r    *requester
	outc chan *MachineStateOutput
}

// MachineStateOutput is the output from the machine_state action
type MachineStateOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// MachineStateResult is the result from a machine_state action
type MachineStateResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*MachineStateOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *MachineStateResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *MachineStateResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *MachineStateOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *MachineStateOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *MachineStateOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseMachineStateOutput parses the result value from the MachineState action into target
func (d *MachineStateOutput) ParseMachineStateOutput(target any) error {
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
func (d *MachineStateRequester) Do(ctx context.Context) (*MachineStateResult, error) {
	dres := &MachineStateResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &MachineStateOutput{
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
func (d *MachineStateResult) AllOutputs() []*MachineStateOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *MachineStateResult) EachOutput(h func(r *MachineStateOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Instance is an optional input to the machine_state action
//
// Description: Machine Instance ID
func (d *MachineStateRequester) Instance(v string) *MachineStateRequester {
	d.r.args["instance"] = v

	return d
}

// Name is an optional input to the machine_state action
//
// Description: Machine Name
func (d *MachineStateRequester) Name(v string) *MachineStateRequester {
	d.r.args["name"] = v

	return d
}

// Path is an optional input to the machine_state action
//
// Description: Machine Path
func (d *MachineStateRequester) Path(v string) *MachineStateRequester {
	d.r.args["path"] = v

	return d
}

// AvailableTransitions is the value of the available_transitions output
//
// Description: The list of available transitions this autonomous agent can make
func (d *MachineStateOutput) AvailableTransitions() []any {
	val := d.reply["available_transitions"]

	return val.([]any)

}

// CurrentState is the value of the current_state output
//
// Description: The Choria Scout specific state for Scout checks
func (d *MachineStateOutput) CurrentState() any {
	val, ok := d.reply["current_state"]
	if !ok || val == nil {
		// we have to avoid returning nil.(any)
		return nil
	}

	return val

}

// Id is the value of the id output
//
// Description: The unique running ID of the autonomous agent
func (d *MachineStateOutput) Id() string {
	val := d.reply["id"]

	return val.(string)

}

// Name is the value of the name output
//
// Description: The name of the autonomous agent
func (d *MachineStateOutput) Name() string {
	val := d.reply["name"]

	return val.(string)

}

// Path is the value of the path output
//
// Description: The location on disk where the autonomous agent is stored
func (d *MachineStateOutput) Path() string {
	val := d.reply["path"]

	return val.(string)

}

// Scout is the value of the scout output
//
// Description: True when this autonomous agent represents a Choria Scout Check
func (d *MachineStateOutput) Scout() bool {
	val := d.reply["scout"]

	return val.(bool)

}

// StartTime is the value of the start_time output
//
// Description: The time the autonomous agent was started in unix seconds
func (d *MachineStateOutput) StartTime() string {
	val := d.reply["start_time"]

	return val.(string)

}

// State is the value of the state output
//
// Description: The current state the agent is in
func (d *MachineStateOutput) State() string {
	val := d.reply["state"]

	return val.(string)

}

// Version is the value of the version output
//
// Description: The version of the autonomous agent
func (d *MachineStateOutput) Version() string {
	val := d.reply["version"]

	return val.(string)

}
