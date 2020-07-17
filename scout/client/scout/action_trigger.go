// generated code; DO NOT EDIT; 2020-07-17 16:25:13.780889 +0200 CEST m=+0.034818411"
//
// Client for Choria RPC Agent 'scout'' Version 0.0.1 generated using Choria version 0.14.0

package scoutclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
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
	stats   *rpcclient.Stats
	outputs []*TriggerOutput
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
	dres := &TriggerResult{}

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

		if d.outc != nil {
			d.outc <- output
			return
		}

		dres.outputs = append(dres.outputs, output)
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
