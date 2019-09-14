package external

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	agentddl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"github.com/sirupsen/logrus"
)

const (
	requestProtocol    = "choria:mcorpc:external_request:1"
	activationProtocol = "choria:mcorpc:external_activation_check:1"
)

// ActivationCheck is the request to determine if an agent should activate
type ActivationCheck struct {
	Protocol string `json:"protocol"`
	Agent    string `json:"agent"`
}

// ActivationReply is the reply from the activation check message
type ActivationReply struct {
	ShouldActivate bool `json:"activate"`
}

// Request is the request being published to the shim runner
type Request struct {
	Protocol   string       `json:"protocol"`
	Agent      string       `json:"agent"`
	Action     string       `json:"action"`
	RequestID  string       `json:"requestid"`
	SenderID   string       `json:"senderid"`
	CallerID   string       `json:"callerid"`
	Collective string       `json:"collective"`
	TTL        int          `json:"ttl"`
	Time       int64        `json:"msgtime"`
	Body       *RequestBody `json:"body"`
}

// RequestBody is the body passed to the
type RequestBody struct {
	Agent  string          `json:"agent"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	Caller string          `json:"caller"`
}

func (p *Provider) newExternalAgent(ddl *agentddl.DDL, mgr server.AgentManager) (*mcorpc.Agent, error) {
	agent := mcorpc.New(ddl.Metadata.Name, ddl.Metadata, mgr.Choria(), mgr.Logger())
	activator, err := p.externalActivationCheck(ddl)
	if err != nil {
		return nil, fmt.Errorf("could not activation check %s: %s", agent.Name(), err)
	}
	agent.SetActivationChecker(activator)

	p.log.Debugf("Registering proxy actions for External agent %s: %s", ddl.Metadata.Name, strings.Join(ddl.ActionNames(), ", "))

	for _, action := range ddl.Actions {
		if err != nil {
			return nil, err
		}

		agent.MustRegisterAction(action.Name, p.externalAction)
	}

	return agent, nil
}

func (p *Provider) externalActivationCheck(ddl *agent.DDL) (mcorpc.ActivationChecker, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	agentPath := filepath.Join(p.dir, ddl.Metadata.Name)
	rep := &ActivationReply{}
	req := &ActivationCheck{
		Protocol: activationProtocol,
		Agent:    ddl.Metadata.Name,
	}

	j, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not json encode activation message: %s", err)
	}

	err = p.executeRequest(ctx, agentPath, activationProtocol, j, rep, p.log)
	if err != nil {
		p.log.Warnf("External agent %s not activating due to error during activation check: %s", agentPath, err)
		return func() bool { return false }, nil
	}

	return func() bool { return rep.ShouldActivate }, nil
}

func (p *Provider) externalAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	action := fmt.Sprintf("%s#%s", req.Agent, req.Action)
	agentPath := filepath.Join(p.dir, req.Agent)

	agent.Log.Debugf("Attempting to call external agent %s (%s) with a timeout %d", action, agentPath, agent.Metadata().Timeout)

	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		p.abortAction(fmt.Sprintf("Cannot call external agent %s: agent executable %s was not found", action, agentPath), agent, reply)
		return
	}

	tctx, cancel := context.WithTimeout(ctx, time.Duration(time.Duration(agent.Metadata().Timeout)*time.Second))
	defer cancel()

	externreq, err := p.newExternalRequest(req)
	if err != nil {
		p.abortAction(fmt.Sprintf("Could not call external agent %s: json request creation failed: %s", action, err), agent, reply)
		return
	}

	err = p.executeRequest(tctx, agentPath, requestProtocol, externreq, reply, agent.Log)
	if err != nil {
		p.abortAction(fmt.Sprintf("Could not call external agent %s: :%s", action, err), agent, reply)
		return
	}

	return
}

func (p *Provider) executeRequest(ctx context.Context, command string, protocol string, req []byte, reply interface{}, log *logrus.Entry) error {
	reqfile, err := ioutil.TempFile("", "request")
	if err != nil {
		return fmt.Errorf("could not create request temp file: %s", err)
	}
	defer os.Remove(reqfile.Name())

	repfile, err := ioutil.TempFile("", "reply")
	if err != nil {
		return fmt.Errorf("could not create reply temp file: %s", err)
	}
	defer os.Remove(repfile.Name())
	repfile.Close()

	_, err = reqfile.Write(req)
	if err != nil {
		return fmt.Errorf("could not create reply temp file: %s", err)
	}

	execution := exec.CommandContext(ctx, command, reqfile.Name(), repfile.Name(), requestProtocol)
	execution.Dir = os.TempDir()
	execution.Env = []string{
		"CHORIA_EXTERNAL_REQUEST=" + reqfile.Name(),
		"CHORIA_EXTERNAL_REPLY=" + repfile.Name(),
		"CHORIA_EXTERNAL_PROTOCOL=" + protocol,
	}

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not open STDOUT: %s", err)
	}

	stderr, err := execution.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not open STDERR: %s", err)
	}

	wg := &sync.WaitGroup{}
	outputReader := func(wg *sync.WaitGroup, in io.ReadCloser, logger func(args ...interface{})) {
		defer wg.Done()

		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			logger(scanner.Text())
		}
	}

	wg.Add(1)
	go outputReader(wg, stderr, log.Error)
	wg.Add(1)
	go outputReader(wg, stdout, log.Info)

	err = execution.Start()
	if err != nil {
		return fmt.Errorf("executing %s failed: %s", filepath.Base(command), err)
	}

	execution.Wait()
	wg.Wait()

	if execution.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("executing %s failed: exit status %d", filepath.Base(command), execution.ProcessState.ExitCode())
	}

	repjson, err := ioutil.ReadFile(repfile.Name())
	if err != nil {
		return fmt.Errorf("failed to read reply json: %s", err)
	}

	err = json.Unmarshal(repjson, reply)
	if err != nil {
		return fmt.Errorf("failed to decode reply json: %s", err)
	}

	return nil
}

func (p *Provider) newExternalRequest(req *mcorpc.Request) ([]byte, error) {
	sr := Request{
		Protocol: "choria:mcorpc:external_request:1",
		Action:   req.Action,
		Agent:    req.Agent,
		Body: &RequestBody{
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

func (p *Provider) abortAction(reason string, agent *mcorpc.Agent, reply *mcorpc.Reply) {
	agent.Log.Error(reason)
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = reason
}
