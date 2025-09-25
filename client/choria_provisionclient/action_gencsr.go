// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_provision' Version 0.29.4 generated using Choria version 0.29.4

package choria_provisionclient

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

// GencsrRequester performs a RPC request to choria_provision#gencsr
type GencsrRequester struct {
	r    *requester
	outc chan *GencsrOutput
}

// GencsrOutput is the output from the gencsr action
type GencsrOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// GencsrResult is the result from a gencsr action
type GencsrResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*GencsrOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *GencsrResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *GencsrResult) Stats() Stats {
	return d.stats
}

// RPCClientStats is the rpc request stats
func (d *GencsrResult) RPCClientStats() *rpcclient.Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *GencsrOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *GencsrOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *GencsrOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseGencsrOutput parses the result value from the Gencsr action into target
func (d *GencsrOutput) ParseGencsrOutput(target any) error {
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
func (d *GencsrRequester) Do(ctx context.Context) (*GencsrResult, error) {
	dres := &GencsrResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &GencsrOutput{
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
func (d *GencsrResult) AllOutputs() []*GencsrOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *GencsrResult) EachOutput(h func(r *GencsrOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// C is an optional input to the gencsr action
//
// Description: Country Code
func (d *GencsrRequester) C(v string) *GencsrRequester {
	d.r.args["c"] = v

	return d
}

// L is an optional input to the gencsr action
//
// Description: Locality or municipality (such as city or town name)
func (d *GencsrRequester) L(v string) *GencsrRequester {
	d.r.args["l"] = v

	return d
}

// O is an optional input to the gencsr action
//
// Description: Organization
func (d *GencsrRequester) O(v string) *GencsrRequester {
	d.r.args["o"] = v

	return d
}

// Ou is an optional input to the gencsr action
//
// Description: Organizational Unit
func (d *GencsrRequester) Ou(v string) *GencsrRequester {
	d.r.args["ou"] = v

	return d
}

// St is an optional input to the gencsr action
//
// Description: State
func (d *GencsrRequester) St(v string) *GencsrRequester {
	d.r.args["st"] = v

	return d
}

// Cn is an optional input to the gencsr action
//
// Description: The certificate Common Name to place in the CSR
func (d *GencsrRequester) Cn(v string) *GencsrRequester {
	d.r.args["cn"] = v

	return d
}

// Csr is the value of the csr output
//
// Description: PEM text block for the CSR
func (d *GencsrOutput) Csr() string {
	val := d.reply["csr"]

	return val.(string)

}

// PublicKey is the value of the public_key output
//
// Description: PEM text block of the public key that made the CSR
func (d *GencsrOutput) PublicKey() string {
	val := d.reply["public_key"]

	return val.(string)

}

// Ssldir is the value of the ssldir output
//
// Description: SSL directory as determined by the server
func (d *GencsrOutput) Ssldir() string {
	val := d.reply["ssldir"]

	return val.(string)

}
