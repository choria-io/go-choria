// generated code; DO NOT EDIT"
//
// Client for Choria RPC Agent 'aaa_signer' Version 0.23.0 generated using Choria version 0.23.0

package aaa_signerclient

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

// SignRequester performs a RPC request to aaa_signer#sign
type SignRequester struct {
	r    *requester
	outc chan *SignOutput
}

// SignOutput is the output from the sign action
type SignOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// SignResult is the result from a sign action
type SignResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*SignOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *SignResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *SignResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *SignOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *SignOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *SignOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseSignOutput parses the result value from the Sign action into target
func (d *SignOutput) ParseSignOutput(target interface{}) error {
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
func (d *SignRequester) Do(ctx context.Context) (*SignResult, error) {
	dres := &SignResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &SignOutput{
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
func (d *SignResult) EachOutput(h func(r *SignOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// SecureRequest is the value of the secure_request output
//
// Description: The signed Secure Request
func (d *SignOutput) SecureRequest() string {
	val := d.reply["secure_request"]

	return val.(string)

}
