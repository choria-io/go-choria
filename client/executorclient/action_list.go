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

// ListRequester performs a RPC request to executor#list
type ListRequester struct {
	r    *requester
	outc chan *ListOutput
}

// ListOutput is the output from the list action
type ListOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// ListResult is the result from a list action
type ListResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*ListOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *ListResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *ListResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *ListOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *ListOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ListOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseListOutput parses the result value from the List action into target
func (d *ListOutput) ParseListOutput(target any) error {
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
func (d *ListRequester) Do(ctx context.Context) (*ListResult, error) {
	dres := &ListResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &ListOutput{
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
func (d *ListResult) AllOutputs() []*ListOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *ListResult) EachOutput(h func(r *ListOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Action is an optional input to the list action
//
// Description: The action that created a job
func (d *ListRequester) Action(v string) *ListRequester {
	d.r.args["action"] = v

	return d
}

// Agent is an optional input to the list action
//
// Description: The agent that create a job
func (d *ListRequester) Agent(v string) *ListRequester {
	d.r.args["agent"] = v

	return d
}

// Before is an optional input to the list action
//
// Description: Unix timestamp to limit jobs on
func (d *ListRequester) Before(v int64) *ListRequester {
	d.r.args["before"] = v

	return d
}

// Caller is an optional input to the list action
//
// Description: The caller id that created a job
func (d *ListRequester) Caller(v string) *ListRequester {
	d.r.args["caller"] = v

	return d
}

// Command is an optional input to the list action
//
// Description: The command that was executed
func (d *ListRequester) Command(v string) *ListRequester {
	d.r.args["command"] = v

	return d
}

// Completed is an optional input to the list action
//
// Description: Limit to jobs that were completed
func (d *ListRequester) Completed(v bool) *ListRequester {
	d.r.args["completed"] = v

	return d
}

// Identity is an optional input to the list action
//
// Description: The host identity that created the job
func (d *ListRequester) Identity(v string) *ListRequester {
	d.r.args["identity"] = v

	return d
}

// Requestid is an optional input to the list action
//
// Description: The Request ID that created the job
func (d *ListRequester) Requestid(v string) *ListRequester {
	d.r.args["requestid"] = v

	return d
}

// Running is an optional input to the list action
//
// Description: Limits to running jobs
func (d *ListRequester) Running(v bool) *ListRequester {
	d.r.args["running"] = v

	return d
}

// Since is an optional input to the list action
//
// Description: Unix timestamp to limit jobs on
func (d *ListRequester) Since(v int64) *ListRequester {
	d.r.args["since"] = v

	return d
}

// Jobs is the value of the jobs output
//
// Description: List of matched jobs
func (d *ListOutput) Jobs() map[string]any {
	val := d.reply["jobs"]

	return val.(map[string]any)

}
