// generated code; DO NOT EDIT"
//
// Client for Choria RPC Agent 'rpcutil' Version 0.24.0 generated using Choria version 0.24.1

package rpcutilclient

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

// DaemonStatsRequester performs a RPC request to rpcutil#daemon_stats
type DaemonStatsRequester struct {
	r    *requester
	outc chan *DaemonStatsOutput
}

// DaemonStatsOutput is the output from the daemon_stats action
type DaemonStatsOutput struct {
	details *ResultDetails
	reply   map[string]interface{}
}

// DaemonStatsResult is the result from a daemon_stats action
type DaemonStatsResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*DaemonStatsOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *DaemonStatsResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *DaemonStatsResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *DaemonStatsOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *DaemonStatsOutput) HashMap() map[string]interface{} {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *DaemonStatsOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseDaemonStatsOutput parses the result value from the DaemonStats action into target
func (d *DaemonStatsOutput) ParseDaemonStatsOutput(target interface{}) error {
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
func (d *DaemonStatsRequester) Do(ctx context.Context) (*DaemonStatsResult, error) {
	dres := &DaemonStatsResult{ddl: d.r.client.ddl}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &DaemonStatsOutput{
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
func (d *DaemonStatsResult) EachOutput(h func(r *DaemonStatsOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Agents is the value of the agents output
//
// Description: List of agents loaded
func (d *DaemonStatsOutput) Agents() []interface{} {
	val := d.reply["agents"]

	return val.([]interface{})

}

// Configfile is the value of the configfile output
//
// Description: Config file used to start the daemon
func (d *DaemonStatsOutput) Configfile() string {
	val := d.reply["configfile"]

	return val.(string)

}

// Filtered is the value of the filtered output
//
// Description: Count of message that didn't pass filter checks
func (d *DaemonStatsOutput) Filtered() int64 {
	val := d.reply["filtered"]

	return val.(int64)

}

// Passed is the value of the passed output
//
// Description: Count of messages that passed filter checks
func (d *DaemonStatsOutput) Passed() int64 {
	val := d.reply["passed"]

	return val.(int64)

}

// Pid is the value of the pid output
//
// Description: Process ID of the Choria Server
func (d *DaemonStatsOutput) Pid() int64 {
	val := d.reply["pid"]

	return val.(int64)

}

// Replies is the value of the replies output
//
// Description: Count of replies sent back to clients
func (d *DaemonStatsOutput) Replies() int64 {
	val := d.reply["replies"]

	return val.(int64)

}

// Starttime is the value of the starttime output
//
// Description: Time the Choria Server started in unix seconds
func (d *DaemonStatsOutput) Starttime() int64 {
	val := d.reply["starttime"]

	return val.(int64)

}

// Threads is the value of the threads output
//
// Description: List of threads active in the Choria Server
func (d *DaemonStatsOutput) Threads() []interface{} {
	val := d.reply["threads"]

	return val.([]interface{})

}

// Times is the value of the times output
//
// Description: Processor time consumed by the Choria Server
func (d *DaemonStatsOutput) Times() map[string]interface{} {
	val := d.reply["times"]

	return val.(map[string]interface{})

}

// Total is the value of the total output
//
// Description: Count of messages received by the Choria Server
func (d *DaemonStatsOutput) Total() int64 {
	val := d.reply["total"]

	return val.(int64)

}

// Ttlexpired is the value of the ttlexpired output
//
// Description: Count of messages that did pass TTL checks
func (d *DaemonStatsOutput) Ttlexpired() int64 {
	val := d.reply["ttlexpired"]

	return val.(int64)

}

// Unvalidated is the value of the unvalidated output
//
// Description: Count of messages that failed security validation
func (d *DaemonStatsOutput) Unvalidated() int64 {
	val := d.reply["unvalidated"]

	return val.(int64)

}

// Validated is the value of the validated output
//
// Description: Count of messages that passed security validation
func (d *DaemonStatsOutput) Validated() int64 {
	val := d.reply["validated"]

	return val.(int64)

}

// Version is the value of the version output
//
// Description: Choria Server Version
func (d *DaemonStatsOutput) Version() string {
	val := d.reply["version"]

	return val.(string)

}
