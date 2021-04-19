// generated code; DO NOT EDIT"
//
// Client for Choria RPC Agent 'choria_provision' Version 0.0.1 generated using Choria version 0.21.0

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

// JwtRequester performs a RPC request to choria_provision#jwt
type JwtRequester struct {
	r    *requester
	outc chan *JwtOutput
}

// JwtOutput is the output from the jwt action
type JwtOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// JwtResult is the result from a jwt action
type JwtResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*JwtOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *JwtResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *JwtResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *JwtOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *JwtOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *JwtOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseOutput parses the result value from the Jwt action into target
func (d *JwtOutput) ParseJwtOutput(target interface{}) error {
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
func (d *JwtRequester) Do(ctx context.Context) (*JwtResult, error) {
	dres := &JwtResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &JwtOutput{
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
func (d *JwtResult) EachOutput(h func(r *JwtOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Token is an optional input to the jwt action
//
// Description: Authentication token to pass to the server
func (d *JwtRequester) Token(v string) *JwtRequester {
	d.r.args["token"] = v

	return d
}

// Jwt is the value of the jwt output
//
// Description: The contents of the JWT token
func (d *JwtOutput) Jwt() interface{} {
	val, ok := d.reply["jwt"]
	if !ok || val == nil {
		// we have to avoid returning nil.(interface{})
		return nil
	}

	return val.(interface{})
}
