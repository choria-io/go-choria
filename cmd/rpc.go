package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-client/client"
	"github.com/choria-io/go-client/discovery/broadcast"
	"github.com/choria-io/go-protocol/protocol"
	rpcClient "github.com/choria-io/mcorpc-agent-provider/mcorpc/client"
	ddl "github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
	"os"
	"strings"
	"sync"
	"time"
)

type jsonDisplay struct {
	rpcClient.RPCReply

	Agent  string `json:"agent"`
	Action string `json:"action"`
	Sender string `json:"sender"`
}

type rpcCommand struct {
	command

	noResults bool

	one        bool
	batch      int
	batchSleep int
	limitSeed  string
	limit      string
	json       bool
	display    string
	config     string
	verbose    bool

	target            string
	discoveryTimeout  int
	timeout           int
	quiet             bool
	ttl               int
	replyTo           string
	discMethod        string
	discOptions       []string
	nodes             string
	publishTimeout    int
	threaded          bool
	sort              bool
	connectionTimeout int

	with         []string
	sel          []string
	withFact     []string
	withClass    []string
	withAgent    []string
	withIdentity []string

	agent  string
	action string
	args   map[string]string
}

func (r *rpcCommand) Setup() (err error) {
	r.args = make(map[string]string)
	r.cmd = cli.app.Command("rpc", "Generic RPC agent client application")

	// TODO: implement me
	//r.cmd.Flag("no-results", "Do not process results, just send request").BoolVar(&r.noResults)

	// TODO: implement me
	//r.cmd.Flag("one", "Send request to only one discovered nodes").Short('1').BoolVar(&r.one)
	r.cmd.Flag("batch", "Do requests in batches").PlaceHolder("SIZE").IntVar(&r.batch)
	r.cmd.Flag("batch-sleep", "Sleep time between batches").IntVar(&r.batchSleep)
	// TODO: implement me
	//r.cmd.Flag("limit-seed", "Seed value for deterministic random batching").StringVar(&r.limitSeed)
	// TODO: implement me
	//r.cmd.Flag("limit", "Send request to only a subset of nodes, can be a percentage").StringVar(&r.limit)
	// json is default and only means of output at this time (and perhaps forever)
	//r.cmd.Flag("json", "Produce JSON output").Short('j').BoolVar(&r.json)
	r.cmd.Flag("display", "Influence how results are displayed. One of ok, all or failed").StringVar(&r.display)
	r.cmd.Flag("verbose", "Be verbose").Short('v').BoolVar(&r.verbose)

	r.cmd.Flag("target", "Target messages to a specific sub collective").Short('T').StringVar(&r.target)
	r.cmd.Flag("discovery-timeout", "Timeout for doing discovery").IntVar(&r.discoveryTimeout)
	r.cmd.Flag("timeout", "Timeout for calling remote agents").Short('t').IntVar(&r.timeout)
	r.cmd.Flag("quiet", "Do not be verbose").Short('q').BoolVar(&r.quiet)
	r.cmd.Flag("ttl", "Set the message validity period").IntVar(&r.ttl)
	r.cmd.Flag("reply-to", "Set a custom target for replies").StringVar(&r.replyTo)
	r.cmd.Flag("disc-method", "Which discovery method to use").StringVar(&r.discMethod)
	r.cmd.Flag("disc-option", "Options to pass to the discovery method").StringsVar(&r.discOptions)
	// TODO: implement me
	//r.cmd.Flag("nodes", "File with list of nodes to address").PlaceHolder("FILE").StringVar(&r.nodes)
	r.cmd.Flag("publish_timeout", "Timeout for publishing requests to remote agents").IntVar(&r.publishTimeout)
	r.cmd.Flag("threaded", "Start publishing requests and receiving responses in threaded mode").BoolVar(&r.threaded)
	// TODO: implement me
	//r.cmd.Flag("sort", "Sort the output of an RPC call before processing").BoolVar(&r.sort)
	r.cmd.Flag("connection-timeout", "Set the timeout for establishing a connection to the middleware").IntVar(&r.connectionTimeout)

	r.cmd.Flag("with", "Combined classes and facts filter").Short('W').StringsVar(&r.with)
	r.cmd.Flag("select", "Compound filter combining facts and classes").Short('S').StringsVar(&r.sel)
	r.cmd.Flag("with-fact", "Match hosts with a certain fact").Short('F').StringsVar(&r.withFact)
	r.cmd.Flag("with-class", "Match hosts with a certain config management class").Short('C').StringsVar(&r.withClass)
	r.cmd.Flag("with-agent", "Match hosts with a certain agent").Short('A').StringsVar(&r.withAgent)
	r.cmd.Flag("with-identity", "Match hosts with a certain configured identity").Short('I').StringsVar(&r.withIdentity)

	r.cmd.Arg("agent", "Agent to call").Required().StringVar(&r.agent)
	r.cmd.Arg("action", "Action to call").Required().StringVar(&r.action)
	r.cmd.Arg("key=val", "Arguments to pass to agent").StringMapVar(&r.args)

	return nil
}

func (r *rpcCommand) Configure() error {

	err = commonConfigure()
	if err != nil {
		return err
	}

	if r.threaded {
		cfg.Threaded = true
	}
	if r.target != "" {
		cfg.MainCollective = r.target
	}
	if r.ttl != 0 {
		cfg.TTL = r.ttl
	}
	if r.discoveryTimeout != 0 {
		cfg.DiscoveryTimeout = r.discoveryTimeout
	}
	if r.discMethod != "" {
		cfg.DefaultDiscoveryMethod = r.discMethod
	}
	if r.discOptions != nil {
		cfg.DefaultDiscoveryOptions = r.discOptions
	}
	if r.publishTimeout != 0 {
		cfg.PublishTimeout = r.publishTimeout
	}
	if r.connectionTimeout != 0 {
		cfg.ConnectionTimeout = r.connectionTimeout
	}

	return nil
}

func (r *rpcCommand) rpcFilter() (*protocol.Filter, error) {
	return client.NewFilter(
		client.FactFilter(r.withFact...),
		client.AgentFilter(r.withAgent...),
		client.ClassFilter(r.withClass...),
		client.IdentityFilter(r.withIdentity...),
		client.CombinedFilter(r.sel...),
		client.CompoundFilter(r.with...),
	)
}

func (r *rpcCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	fw, err := choria.NewWithConfig(cfg)
	if err != nil {
		return err
	}

	// looks for json file
	addl, err := ddl.Find(r.agent, cfg.LibDir)
	if err != nil {
		return err
	}

	rpc, err := rpcClient.New(fw, r.agent, rpcClient.DDL(addl))
	if err != nil {
		return err
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if r.timeout != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(r.timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	if !r.quiet {
		fmt.Print("\nPerforming discovery...")
	}
	b := broadcast.New(fw)

	filters, err := r.rpcFilter()
	if err != nil {
		return err
	}
	nodes, err := b.Discover(ctx, broadcast.Filter(filters))
	if err != nil {
		return err
	}

	if !r.quiet {
		fmt.Printf("Number of nodes: %d\n\n", len(nodes))
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes available to send requests to")
	}

	replies := make(chan jsonDisplay, len(nodes))

	handler := rpcClient.ReplyHandler(func(reply protocol.Reply, rpcr *rpcClient.RPCReply) {
		display := &jsonDisplay{}
		err := json.Unmarshal([]byte(reply.Message()), display)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to unmarshal incoming message from %s", reply.SenderID())
			return
		}
		display.Agent = r.agent
		display.Action = r.action
		display.Sender = reply.SenderID()
		replies <- *display
	})

	doArgs := []rpcClient.RequestOption{rpcClient.Workers(1), rpcClient.Targets(nodes), handler}

	if r.batchSleep != 0 || r.batch != 0 {
		if r.batchSleep != 0 && r.batch != 0 {
			doArgs = append(doArgs, rpcClient.InBatches(r.batch, r.batchSleep))
		} else {
			return fmt.Errorf("you must set both --batch=<batchSize> *and* --batchSleep=<sleepTime>")
		}
	}
	if r.replyTo != "" {
		doArgs = append(doArgs, rpcClient.ReplyTo(r.replyTo))
	}

	resp, err := rpc.Do(ctx, r.action, r.args, doArgs...)
	if err != nil {
		return err
	}

	var displays []jsonDisplay
	for range nodes {
		var display jsonDisplay
		select {
		case display = <-replies:
			displays = append(displays, display)
		default:
			// nothing in the channel prematurely, means at least one reply errored when unmarshalling or not all messages were received
			break
		}
	}

	dataEncoded, err := json.Marshal(displays)
	if err != nil {
		return err
	}

	var outBuf bytes.Buffer
	if err = json.Indent(&outBuf, dataEncoded, "", "\t"); err != nil {
		return err
	}

	if _, err = outBuf.WriteTo(os.Stdout); err != nil {
		return err
	}

	stat := resp.Stats()
	if len(stat.NoResponseFrom()) > 0 {
		_, err := fmt.Fprintf(os.Stderr, "Did not get any responses from: %s", strings.Join(stat.NoResponseFrom(), ", "))
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	cli.commands = append(cli.commands, &rpcCommand{})
}
