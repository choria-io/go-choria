// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'scout' Version 0.29.4 generated using Choria version 0.29.4

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

// GossValidateRequester performs a RPC request to scout#goss_validate
type GossValidateRequester struct {
	r    *requester
	outc chan *GossValidateOutput
}

// GossValidateOutput is the output from the goss_validate action
type GossValidateOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// GossValidateResult is the result from a goss_validate action
type GossValidateResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*GossValidateOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *GossValidateResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *GossValidateResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *GossValidateOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *GossValidateOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *GossValidateOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseGossValidateOutput parses the result value from the GossValidate action into target
func (d *GossValidateOutput) ParseGossValidateOutput(target any) error {
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
func (d *GossValidateRequester) Do(ctx context.Context) (*GossValidateResult, error) {
	dres := &GossValidateResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &GossValidateOutput{
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
func (d *GossValidateResult) AllOutputs() []*GossValidateOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *GossValidateResult) EachOutput(h func(r *GossValidateOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// File is an optional input to the goss_validate action
//
// Description: Path to the Goss validation specification
func (d *GossValidateRequester) File(v string) *GossValidateRequester {
	d.r.args["file"] = v

	return d
}

// Vars is an optional input to the goss_validate action
//
// Description: Path to a file to use as template variables
func (d *GossValidateRequester) Vars(v string) *GossValidateRequester {
	d.r.args["vars"] = v

	return d
}

// YamlRules is an optional input to the goss_validate action
//
// Description: Contents of the Gossfile to validate
func (d *GossValidateRequester) YamlRules(v string) *GossValidateRequester {
	d.r.args["yaml_rules"] = v

	return d
}

// YamlVars is an optional input to the goss_validate action
//
// Description: YAML data to use as variables
func (d *GossValidateRequester) YamlVars(v string) *GossValidateRequester {
	d.r.args["yaml_vars"] = v

	return d
}

// Failures is the value of the failures output
//
// Description: The number of tests that failed
func (d *GossValidateOutput) Failures() int64 {
	val := d.reply["failures"]

	return val.(int64)

}

// Results is the value of the results output
//
// Description: The full test results
func (d *GossValidateOutput) Results() []any {
	val := d.reply["results"]

	return val.([]any)

}

// Runtime is the value of the runtime output
//
// Description: The time it took to run the tests, in seconds
func (d *GossValidateOutput) Runtime() int64 {
	val := d.reply["runtime"]

	return val.(int64)

}

// Skipped is the value of the skipped output
//
// Description: Indicates how many tests were skipped
func (d *GossValidateOutput) Skipped() int64 {
	val := d.reply["skipped"]

	return val.(int64)

}

// Success is the value of the success output
//
// Description: Indicates how many tests passed
func (d *GossValidateOutput) Success() int64 {
	val := d.reply["success"]

	return val.(int64)

}

// Summary is the value of the summary output
//
// Description: A human friendly test result
func (d *GossValidateOutput) Summary() string {
	val := d.reply["summary"]

	return val.(string)

}

// Tests is the value of the tests output
//
// Description: The number of tests that were run
func (d *GossValidateOutput) Tests() int64 {
	val := d.reply["tests"]

	return val.(int64)

}
