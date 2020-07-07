// generated code; DO NOT EDIT; 2020-07-07 17:35:21.439397 +0200 CEST m=+0.019895169"
//
// Client for Choria RPC Agent 'scout'' Version 0.0.1 generated using Choria version 0.14.0

package scoutclient

import (
	"context"
	"encoding/json"

	"github.com/choria-io/go-choria/protocol"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
)

// ChecksRequester performs a RPC request to scout#checks
type ChecksRequester struct {
	r    *requester
	outc chan *ChecksOutput
}

// ChecksOutput is the output from the checks action
type ChecksOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// ChecksResult is the result from a checks action
type ChecksResult struct {
	stats   *rpcclient.Stats
	outputs []*ChecksOutput
}

// Stats is the rpc request stats
func (d *ChecksResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *ChecksOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *ChecksOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ChecksOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// Do performs the request
func (d *ChecksRequester) Do(ctx context.Context) (*ChecksResult, error) {
	dres := &ChecksResult{}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		output := &ChecksOutput{
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
func (d *ChecksResult) EachOutput(h func(r *ChecksOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Checks is the value of the checks output
//
// Description: Details about each check
func (d *ChecksOutput) Checks() []interface{} {
	val := d.reply["checks"]
	return val.([]interface{})
}
