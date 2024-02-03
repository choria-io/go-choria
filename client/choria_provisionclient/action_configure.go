// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'choria_provision' Version 0.28.0 generated using Choria version 0.28.0

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

// ConfigureRequester performs a RPC request to choria_provision#configure
type ConfigureRequester struct {
	r    *requester
	outc chan *ConfigureOutput
}

// ConfigureOutput is the output from the configure action
type ConfigureOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// ConfigureResult is the result from a configure action
type ConfigureResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*ConfigureOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *ConfigureResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *ConfigureResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *ConfigureOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *ConfigureOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *ConfigureOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseConfigureOutput parses the result value from the Configure action into target
func (d *ConfigureOutput) ParseConfigureOutput(target any) error {
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
func (d *ConfigureRequester) Do(ctx context.Context) (*ConfigureResult, error) {
	dres := &ConfigureResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &ConfigureOutput{
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
func (d *ConfigureResult) AllOutputs() []*ConfigureOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *ConfigureResult) EachOutput(h func(r *ConfigureOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// ActionPolicies is an optional input to the configure action
//
// Description: Map of Action Policy documents indexed by file name
func (d *ConfigureRequester) ActionPolicies(v map[string]any) *ConfigureRequester {
	d.r.args["action_policies"] = v

	return d
}

// Ca is an optional input to the configure action
//
// Description: PEM text block for the CA
func (d *ConfigureRequester) Ca(v string) *ConfigureRequester {
	d.r.args["ca"] = v

	return d
}

// Certificate is an optional input to the configure action
//
// Description: PEM text block for the certificate
func (d *ConfigureRequester) Certificate(v string) *ConfigureRequester {
	d.r.args["certificate"] = v

	return d
}

// EcdhPublic is an optional input to the configure action
//
// Description: Required when sending a private key
func (d *ConfigureRequester) EcdhPublic(v string) *ConfigureRequester {
	d.r.args["ecdh_public"] = v

	return d
}

// Key is an optional input to the configure action
//
// Description: A RSA private key
func (d *ConfigureRequester) Key(v string) *ConfigureRequester {
	d.r.args["key"] = v

	return d
}

// OpaPolicies is an optional input to the configure action
//
// Description: Map of Open Policy Agent Policy documents indexed by file name
func (d *ConfigureRequester) OpaPolicies(v map[string]any) *ConfigureRequester {
	d.r.args["opa_policies"] = v

	return d
}

// ServerJwt is an optional input to the configure action
//
// Description: JWT file used to identify the server to the broker for ed25519 based authentication
func (d *ConfigureRequester) ServerJwt(v string) *ConfigureRequester {
	d.r.args["server_jwt"] = v

	return d
}

// Ssldir is an optional input to the configure action
//
// Description: Directory for storing the certificate in
func (d *ConfigureRequester) Ssldir(v string) *ConfigureRequester {
	d.r.args["ssldir"] = v

	return d
}

// Token is an optional input to the configure action
//
// Description: Authentication token to pass to the server
func (d *ConfigureRequester) Token(v string) *ConfigureRequester {
	d.r.args["token"] = v

	return d
}

// Message is the value of the message output
//
// Description: Status message from the Provisioner
func (d *ConfigureOutput) Message() string {
	val := d.reply["message"]

	return val.(string)

}
