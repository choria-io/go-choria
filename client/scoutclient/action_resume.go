// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'scout' Version 0.29.1 generated using Choria version 0.29.3

package scoutclient

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

// ResumeRequester performs a RPC request to scout#resume
type ResumeRequester struct {
	r    *requester
	outc chan *ResumeOutput
}

// ResumeOutput is the output from the resume action
type ResumeOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// ResumeResult is the result from a resume action
type ResumeResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*ResumeOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *ResumeResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *ResumeResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *ResumeOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *ResumeOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ResumeOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseResumeOutput parses the result value from the Resume action into target
func (d *ResumeOutput) ParseResumeOutput(target any) error {
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
func (d *ResumeRequester) Do(ctx context.Context) (*ResumeResult, error) {
	dres := &ResumeResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &ResumeOutput{
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
func (d *ResumeResult) AllOutputs() []*ResumeOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *ResumeResult) EachOutput(h func(r *ResumeOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Checks is an optional input to the resume action
//
// Description: Check to resume, empty means all
func (d *ResumeRequester) Checks(v []any) *ResumeRequester {
	d.r.args["checks"] = v

	return d
}

// Failed is the value of the failed output
//
// Description: List of checks that could not be resumed
func (d *ResumeOutput) Failed() []any {
	val := d.reply["failed"]

	return val.([]any)

}

// Skipped is the value of the skipped output
//
// Description: List of checks that was skipped
func (d *ResumeOutput) Skipped() []any {
	val := d.reply["skipped"]

	return val.([]any)

}

// Transitioned is the value of the transitioned output
//
// Description: List of checks that were resumed
func (d *ResumeOutput) Transitioned() []any {
	val := d.reply["transitioned"]

	return val.([]any)

}
