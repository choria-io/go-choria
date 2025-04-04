// generated code; DO NOT EDIT
//
// Client for Choria RPC Agent 'executor' Version 0.29.4 generated using Choria version 0.29.4

package executorclient

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

// StatusRequester performs a RPC request to executor#status
type StatusRequester struct {
	r    *requester
	outc chan *StatusOutput
}

// StatusOutput is the output from the status action
type StatusOutput struct {
	details *ResultDetails
	reply   map[string]any
}

// StatusResult is the result from a status action
type StatusResult struct {
	ddl        *agent.DDL
	stats      *rpcclient.Stats
	outputs    []*StatusOutput
	rpcreplies []*replyfmt.RPCReply
	mu         sync.Mutex
}

func (d *StatusResult) RenderResults(w io.Writer, format RenderFormat, displayMode DisplayMode, verbose bool, silent bool, colorize bool, log Log) error {
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
func (d *StatusResult) Stats() Stats {
	return d.stats
}

// ResultDetails is the details about the request
func (d *StatusOutput) ResultDetails() *ResultDetails {
	return d.details
}

// HashMap is the raw output data
func (d *StatusOutput) HashMap() map[string]any {
	return d.reply
}

// JSON is the JSON representation of the output data
func (d *StatusOutput) JSON() ([]byte, error) {
	return json.Marshal(d.reply)
}

// ParseStatusOutput parses the result value from the Status action into target
func (d *StatusOutput) ParseStatusOutput(target any) error {
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
func (d *StatusRequester) Do(ctx context.Context) (*StatusResult, error) {
	dres := &StatusResult{ddl: d.r.client.ddl}

	addl, err := dres.ddl.ActionInterface(d.r.action)
	if err != nil {
		return nil, err
	}

	handler := func(pr protocol.Reply, r *rpcclient.RPCReply) {
		// filtered by expr filter
		if r == nil {
			return
		}

		output := &StatusOutput{
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
func (d *StatusResult) AllOutputs() []*StatusOutput {
	return d.outputs
}

// EachOutput iterates over all results received
func (d *StatusResult) EachOutput(h func(r *StatusOutput)) {
	for _, resp := range d.outputs {
		h(resp)
	}
}

// Action is the value of the action output
//
// Description: The RPC Action that started the process
func (d *StatusOutput) Action() string {
	val := d.reply["action"]

	return val.(string)

}

// Agent is the value of the agent output
//
// Description: The RPC Agent that started the process
func (d *StatusOutput) Agent() string {
	val := d.reply["agent"]

	return val.(string)

}

// Args is the value of the args output
//
// Description: The command arguments, if the caller has access
func (d *StatusOutput) Args() string {
	val := d.reply["args"]

	return val.(string)

}

// Caller is the value of the caller output
//
// Description: The Caller ID who started the process
func (d *StatusOutput) Caller() string {
	val := d.reply["caller"]

	return val.(string)

}

// Command is the value of the command output
//
// Description: The command being executed, if the caller has access
func (d *StatusOutput) Command() string {
	val := d.reply["command"]

	return val.(string)

}

// ExitCode is the value of the exit_code output
//
// Description: The exit code the process terminated with
func (d *StatusOutput) ExitCode() int64 {
	val := d.reply["exit_code"]

	return val.(int64)

}

// ExitReason is the value of the exit_reason output
//
// Description: If the process failed, the reason for th failure
func (d *StatusOutput) ExitReason() string {
	val := d.reply["exit_reason"]

	return val.(string)

}

// Pid is the value of the pid output
//
// Description: The OS Process ID
func (d *StatusOutput) Pid() int64 {
	val := d.reply["pid"]

	return val.(int64)

}

// Requestid is the value of the requestid output
//
// Description: The Request ID that started the process
func (d *StatusOutput) Requestid() string {
	val := d.reply["requestid"]

	return val.(string)

}

// Running is the value of the running output
//
// Description: Indicates if the process is still running
func (d *StatusOutput) Running() bool {
	val := d.reply["running"]

	return val.(bool)

}

// StartTime is the value of the start_time output
//
// Description: Time that the process started
func (d *StatusOutput) StartTime() string {
	val := d.reply["start_time"]

	return val.(string)

}

// Started is the value of the started output
//
// Description: Indicates if the process was started
func (d *StatusOutput) Started() bool {
	val := d.reply["started"]

	return val.(bool)

}

// StderrBytes is the value of the stderr_bytes output
//
// Description: The number of bytes of STDERR output available
func (d *StatusOutput) StderrBytes() int64 {
	val := d.reply["stderr_bytes"]

	return val.(int64)

}

// StdoutBytes is the value of the stdout_bytes output
//
// Description: The number of bytes of STDOUT output available
func (d *StatusOutput) StdoutBytes() int64 {
	val := d.reply["stdout_bytes"]

	return val.(int64)

}

// TerminateTime is the value of the terminate_time output
//
// Description: Time that the process terminated
func (d *StatusOutput) TerminateTime() string {
	val := d.reply["terminate_time"]

	return val.(string)

}
