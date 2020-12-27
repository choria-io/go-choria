// generated code; DO NOT EDIT; 2020-12-27 21:55:27.107082 +0100 CET m=+0.089271288"
//
// Client for Choria RPC Agent 'scout'' Version 0.0.1 generated using Choria version 0.18.0

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

// TriggerRequester performs a RPC request to scout#trigger
type TriggerRequester struct {
	r    *requester
	outc chan *TriggerOutput
}

// TriggerOutput is the output from the trigger action
type TriggerOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// TriggerResult is the result from a trigger action
type TriggerResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*TriggerOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *TriggerResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, log Log) error {
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
		return results.RenderTXT(w, addl, verbose, silent, replyfmt.DisplayMode(displayMode), log)
	}
}

// Stats is the rpc request stats
func (d *TriggerResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *TriggerOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *TriggerOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *TriggerOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseOutput parses the result value from the Trigger action into target
func (d *TriggerOutput) ParseTriggerOutput(target interface{}) error {
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
func (d *TriggerRequester) Do(ctx context.Context) (*TriggerResult, error) {
	dres := &TriggerResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		output := &TriggerOutput{
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
func (d *TriggerResult) EachOutput(h func(r *TriggerOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Checks is an optional input to the trigger action
//
// Description: Check to trigger, empty means all
func (d *TriggerRequester) Checks(v []interface{}) *TriggerRequester {
	d.r.args["checks"] = v

	return d
}

// Failed is the value of the failed output
//
// Description: List of checks that could not be triggered
func (d *TriggerOutput) Failed() []interface{} {
	val := d.reply["failed"]
	return val.([]interface{})
}

// Skipped is the value of the skipped output
//
// Description: List of checks that was skipped
func (d *TriggerOutput) Skipped() []interface{} {
	val := d.reply["skipped"]
	return val.([]interface{})
}

// Transitioned is the value of the transitioned output
//
// Description: List of checks that were triggered
func (d *TriggerOutput) Transitioned() []interface{} {
	val := d.reply["transitioned"]
	return val.([]interface{})
}
