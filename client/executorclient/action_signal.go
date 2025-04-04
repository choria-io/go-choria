// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'executor' Version 0.29.4 generated using Choria version 0.29.4

package executorclient

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

// SignalRequester performs a RPC request to executor#signal
type SignalRequester struct {
	r    *requester
	outc chan *SignalOutput
}

// SignalOutput is the output from the signal action
type SignalOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// SignalResult is the result from a signal action
type SignalResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*SignalOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *SignalResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *SignalResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *SignalOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *SignalOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *SignalOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseSignalOutput parses the result value from the Signal action into target
func (d *SignalOutput) ParseSignalOutput(target any) error {
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
func (d *SignalRequester) Do(ctx context.Context) (*SignalResult, error) {
	dres := &SignalResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &SignalOutput{
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
func (d *SignalResult) AllOutputs() []*SignalOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *SignalResult) EachOutput(h func(r *SignalOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Pid is the value of the pid output
//
// Description: The PID that was signaled
func (d *SignalOutput) Pid() int64 {
	val := d.reply["pid"]

	return val.(int64)

}

// Running is the value of the running output
//
// Description: If the process was running after signaling
func (d *SignalOutput) Running() bool {
	val := d.reply["running"]

	return val.(bool)

}
