package ruby

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/mcorpc"
	"github.com/choria-io/go-choria/mcorpc/ddl/agent"
)

// ShimRequest is the request being published to the shim runner
type ShimRequest struct {
	Agent      string           `json:"agent"`
	Action     string           `json:"action"`
	RequestID  string           `json:"requestid"`
	SenderID   string           `json:"senderid"`
	CallerID   string           `json:"callerid"`
	Collective string           `json:"collective"`
	TTL        int              `json:"ttl"`
	Time       int64            `json:"time"`
	Body       *ShimRequestBody `json:"body"`
}

// ShimRequestBody is the body passed to the
type ShimRequestBody struct {
	Agent  string          `json:"agent"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	Caller string          `json:"caller"`
}

// NewRubyAgent creates a shim agent that calls to a old mcollective agent implemented in ruby
func NewRubyAgent(ddl *agent.DDL, mgr AgentManager) (*mcorpc.Agent, error) {
	agent := mcorpc.New(ddl.Metadata.Name, ddl.Metadata, mgr.Choria(), mgr.Logger())

	agent.Log.Debugf("Registering proxy actions for Ruby agent %s: %s", ddl.Metadata.Name, strings.Join(ddl.ActionNames(), ", "))

	for _, action := range ddl.ActionNames() {
		int, err := ddl.ActionInterface(action)
		if err != nil {
			return nil, err
		}

		agent.MustRegisterAction(int.Name, rubyAction)
	}

	return agent, nil
}

func rubyAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	action := fmt.Sprintf("%s#%s", req.Agent, req.Action)
	shim := agent.Config.Choria.RubyAgentShim
	shimcfg := agent.Config.Choria.RubyAgentConfig

	agent.Log.Debugf("Attempting to call Ruby agent %s", action)

	if shim == "" {
		abortAction(fmt.Sprintf("Cannot call Ruby action %s: Ruby compatability shim was not configured", action), agent, reply)
		return
	}

	if shimcfg == "" {
		abortAction(fmt.Sprintf("Cannot call Ruby action %s: Ruby compatability shim configuration file not configured", action), agent, reply)
		return
	}

	if _, err := os.Stat(shim); os.IsNotExist(err) {
		abortAction(fmt.Sprintf("Cannot call Ruby action %s: Ruby compatability shim was not found in %s", action, shim), agent, reply)
		return
	}

	if _, err := os.Stat(shimcfg); os.IsNotExist(err) {
		abortAction(fmt.Sprintf("Cannot call Ruby action %s: Ruby compatability shim configuration file was not found in %s", action, shimcfg), agent, reply)
		return
	}

	tctx, cancel := context.WithTimeout(ctx, time.Duration(time.Duration(agent.Metadata().Timeout)*time.Second))
	defer cancel()

	execution := exec.CommandContext(tctx, agent.Config.Choria.RubyAgentShim, "--config", shimcfg)
	stdin, err := execution.StdinPipe()
	if err != nil {
		abortAction(fmt.Sprintf("Cannot create stdin while calling Ruby action %s: %s", action, err), agent, reply)
		return
	}

	shimr, err := newShimRequest(req)
	if err != nil {
		abortAction(fmt.Sprintf("Cannot prepare request data for Ruby action %s: %s", action, err), agent, reply)
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(shimr))
	}()

	out, err := execution.Output()
	if err != nil {
		abortAction(fmt.Sprintf("Cannot run Ruby action %s: %s", action, err), agent, reply)
		return
	}

	if len(out) < 3 {
		agent.Log.Debugf("No response data received from %s, surpressing replies", action)
		reply.DisableResponse = true
		return
	}

	json.Unmarshal(out, reply)
}

func newShimRequest(req *mcorpc.Request) ([]byte, error) {
	sr := ShimRequest{
		Action: req.Action,
		Agent:  req.Agent,
		Body: &ShimRequestBody{
			Action: req.Action,
			Agent:  req.Agent,
			Caller: req.CallerID,
			Data:   req.Data,
		},
		CallerID:   req.CallerID,
		Collective: req.Collective,
		RequestID:  req.RequestID,
		SenderID:   req.SenderID,
		Time:       req.Time.Unix(),
		TTL:        req.TTL,
	}

	return json.Marshal(sr)
}

func abortAction(reason string, agent *mcorpc.Agent, reply *mcorpc.Reply) {
	agent.Log.Errorf(reason)
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = reason
}
