// generated code; DO NOT EDIT; 2020-07-17 16:25:13.774536 +0200 CEST m=+0.028465376"
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

// ResumeRequester performs a RPC request to scout#resume
type ResumeRequester struct {
	r    *requester
	outc chan *ResumeOutput
}

// ResumeOutput is the output from the resume action
type ResumeOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// ResumeResult is the result from a resume action
type ResumeResult struct {
	stats   *rpcclient.Stats
	outputs []*ResumeOutput
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
func (d *ResumeOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ResumeOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseOutput parses the result value from the Resume action into target
func (d *ResumeOutput) ParseResumeOutput(target interface{}) error {
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
	dres := &ResumeResult{}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		output := &ResumeOutput{
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
func (d *ResumeResult) EachOutput(h func(r *ResumeOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Checks is an optional input to the resume action
//
// Description: Check to resume, empty means all
func (d *ResumeRequester) Checks(v []interface{}) *ResumeRequester {
	d.r.args["checks"] = v

	return d
}

// Failed is the value of the failed output
//
// Description: List of checks that could not be resumed
func (d *ResumeOutput) Failed() []interface{} {
	val := d.reply["failed"]
	return val.([]interface{})
}

// Skipped is the value of the skipped output
//
// Description: List of checks that was skipped
func (d *ResumeOutput) Skipped() []interface{} {
	val := d.reply["skipped"]
	return val.([]interface{})
}

// Transitioned is the value of the transitioned output
//
// Description: List of checks that were resumed
func (d *ResumeOutput) Transitioned() []interface{} {
	val := d.reply["transitioned"]
	return val.([]interface{})
}
