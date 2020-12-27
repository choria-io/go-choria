// generated code; DO NOT EDIT; 2020-12-27 12:48:27.294474 +0100 CET m=+0.050845455"
//
// Client for Choria RPC Agent 'choria_util'' Version 0.19.0 generated using Choria version 0.18.0

package choria_utilclient

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

// InfoRequester performs a RPC request to choria_util#info
type InfoRequester struct {
	r    *requester
	outc chan *InfoOutput
}

// InfoOutput is the output from the info action
type InfoOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// InfoResult is the result from a info action
type InfoResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*InfoOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *InfoResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, log Log) error {
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
func (d *InfoResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *InfoOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *InfoOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *InfoOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseOutput parses the result value from the Info action into target
func (d *InfoOutput) ParseInfoOutput(target interface{}) error {
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
func (d *InfoRequester) Do(ctx context.Context) (*InfoResult, error) {
	dres := &InfoResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		output := &InfoOutput{
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
func (d *InfoResult) EachOutput(h func(r *InfoOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// ChoriaVersion is the value of the choria_version output
//
// Description: Choria version
func (d *InfoOutput) ChoriaVersion() interface{} {
	val := d.reply["choria_version"]
	return val.(interface{})
}

// ClientFlavour is the value of the client_flavour output
//
// Description: Middleware client gem flavour
func (d *InfoOutput) ClientFlavour() interface{} {
	val := d.reply["client_flavour"]
	return val.(interface{})
}

// ClientOptions is the value of the client_options output
//
// Description: Active Middleware client gem options
func (d *InfoOutput) ClientOptions() interface{} {
	val := d.reply["client_options"]
	return val.(interface{})
}

// ClientStats is the value of the client_stats output
//
// Description: Middleware client gem statistics
func (d *InfoOutput) ClientStats() interface{} {
	val := d.reply["client_stats"]
	return val.(interface{})
}

// ClientVersion is the value of the client_version output
//
// Description: Middleware client library version
func (d *InfoOutput) ClientVersion() interface{} {
	val := d.reply["client_version"]
	return val.(interface{})
}

// ConnectedServer is the value of the connected_server output
//
// Description: Connected middleware server
func (d *InfoOutput) ConnectedServer() interface{} {
	val := d.reply["connected_server"]
	return val.(interface{})
}

// Connector is the value of the connector output
//
// Description: Connector plugin
func (d *InfoOutput) Connector() interface{} {
	val := d.reply["connector"]
	return val.(interface{})
}

// ConnectorTls is the value of the connector_tls output
//
// Description: If the connector is running with TLS security enabled
func (d *InfoOutput) ConnectorTls() interface{} {
	val := d.reply["connector_tls"]
	return val.(interface{})
}

// FacterCommand is the value of the facter_command output
//
// Description: Command used for Facter
func (d *InfoOutput) FacterCommand() interface{} {
	val := d.reply["facter_command"]
	return val.(interface{})
}

// FacterDomain is the value of the facter_domain output
//
// Description: Facter domain
func (d *InfoOutput) FacterDomain() interface{} {
	val := d.reply["facter_domain"]
	return val.(interface{})
}

// MiddlewareServers is the value of the middleware_servers output
//
// Description: Middleware Servers configured or discovered
func (d *InfoOutput) MiddlewareServers() interface{} {
	val := d.reply["middleware_servers"]
	return val.(interface{})
}

// Path is the value of the path output
//
// Description: Active OS PATH
func (d *InfoOutput) Path() interface{} {
	val := d.reply["path"]
	return val.(interface{})
}

// SecureProtocol is the value of the secure_protocol output
//
// Description: If the protocol is running with PKI security enabled
func (d *InfoOutput) SecureProtocol() interface{} {
	val := d.reply["secure_protocol"]
	return val.(interface{})
}

// Security is the value of the security output
//
// Description: Security Provider plugin
func (d *InfoOutput) Security() interface{} {
	val := d.reply["security"]
	return val.(interface{})
}

// SrvDomain is the value of the srv_domain output
//
// Description: Configured SRV domain
func (d *InfoOutput) SrvDomain() interface{} {
	val := d.reply["srv_domain"]
	return val.(interface{})
}

// UsingSrv is the value of the using_srv output
//
// Description: Indicates if SRV records are considered
func (d *InfoOutput) UsingSrv() interface{} {
	val := d.reply["using_srv"]
	return val.(interface{})
}
