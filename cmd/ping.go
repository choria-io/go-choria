package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-client/client"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/guptarohit/asciigraph"
	log "github.com/sirupsen/logrus"
)

type pingCommand struct {
	command

	silent     bool
	collective string
	timeout    int
	graph      bool
	workers    int
	waitfor    int
	factF      []string
	agentsF    []string
	classF     []string
	identityF  []string
	combinedF  []string
	namesOnly  bool

	start time.Time
	ctr   int
	mu    *sync.Mutex

	times []float64
}

func (p *pingCommand) Setup() (err error) {
	p.cmd = cli.app.Command("ping", "Low level Choria network testing tool")
	p.cmd.Flag("silent", "Do not print any hostnames").BoolVar(&p.silent)
	p.cmd.Flag("names", "Only show the names that respond, no statistics").BoolVar(&p.namesOnly)
	p.cmd.Flag("wf", "Match hosts with a certain fact").Short('F').StringsVar(&p.factF)
	p.cmd.Flag("wc", "Match hosts with a certain configuration management class").Short('C').StringsVar(&p.classF)
	p.cmd.Flag("wa", "Match hosts with a certain Choria agent").Short('A').StringsVar(&p.agentsF)
	p.cmd.Flag("wi", "Match hosts with a certain Choria identity").Short('I').StringsVar(&p.identityF)
	p.cmd.Flag("with", "Combined classes and facts filter").Short('W').PlaceHolder("FILTER").StringsVar(&p.combinedF)
	p.cmd.Flag("expect", "Wait until this many replies were received or timeout").IntVar(&p.waitfor)
	p.cmd.Flag("target", "Target a specific sub collective").Short('T').StringVar(&p.collective)
	p.cmd.Flag("timeout", "How long to wait for responses").IntVar(&p.timeout)
	p.cmd.Flag("graph", "Produce a graph of the result times").BoolVar(&p.graph)
	p.cmd.Flag("workers", "How many workers to start for receiving messages").Default("3").IntVar(&p.workers)

	return
}

func (p *pingCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	p.times = []float64{}
	p.mu = &sync.Mutex{}

	if p.timeout == 0 {
		p.timeout = cfg.DiscoveryTimeout
	}

	if p.collective == "" {
		p.collective = cfg.MainCollective
	}

	filter, err := client.NewFilter(
		client.FactFilter(p.factF...),
		client.AgentFilter(p.agentsF...),
		client.ClassFilter(p.classF...),
		client.IdentityFilter(p.identityF...),
		client.CombinedFilter(p.combinedF...),
	)
	if err != nil {
		return fmt.Errorf("could not parse filters: %s", err)
	}

	cl, err := client.New(c, client.Receivers(p.workers), client.Timeout(time.Duration(p.timeout)*time.Second))
	if err != nil {
		return fmt.Errorf("could not setup client: %s", err)
	}

	msg, err := p.createMessage(protocol.NewFilter())
	if err != nil {
		return fmt.Errorf("could not create message: %s", err)
	}

	msg.Filter = filter

	p.start = time.Now()

	err = cl.Request(ctx, msg, p.handler)
	if err != nil {
		return fmt.Errorf("could not perform request: %s", err)
	}

	if !p.namesOnly {
		err = p.summarize()
	}

	return err
}

func (p *pingCommand) summarize() error {
	if !p.silent {
		fmt.Printf("\n\n")
	}

	fmt.Println("---- ping statistics ----")

	if len(p.times) > 0 {
		sum := 0.0
		min := 999999.0
		max := -1.0
		avg := 0.0

		for _, value := range p.times {
			sum += value
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}

		avg = sum / float64(len(p.times))

		fmt.Printf("%d replies max: %.2f min: %.2f avg: %.2f\n", len(p.times), max, min, avg)

		if p.graph {
			fmt.Println()
			fmt.Println(p.chart())
			fmt.Println()
		}

		return nil
	}

	return errors.New("no responses received")
}

func (p *pingCommand) handler(ctx context.Context, m *choria.ConnectorMessage) {
	reply, err := c.NewTransportFromJSON(string(m.Data))
	if err != nil {
		log.Errorf("Could not process a reply: %s", err)
		return
	}

	now := time.Now()
	delay := now.Sub(p.start).Seconds() * 1000

	p.mu.Lock()
	defer p.mu.Unlock()

	p.times = append(p.times, delay)

	if !p.silent {
		if p.namesOnly {
			fmt.Println(reply.SenderID())
		} else {
			fmt.Printf("%-40s time=%0.3f ms\n", reply.SenderID(), delay)
		}
	}

	p.ctr++
	if p.waitfor == p.ctr {
		cancel()
	}
}

func (p *pingCommand) createMessage(filter *protocol.Filter) (*choria.Message, error) {
	msg, err := c.NewMessage(base64.StdEncoding.EncodeToString([]byte("ping")), "discovery", p.collective, "request", nil)
	if err != nil {
		return nil, fmt.Errorf("could not create message: %s", err)
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo(choria.ReplyTarget(msg, msg.RequestID))

	msg.Filter = filter

	return msg, err
}

func (p *pingCommand) Configure() error {
	protocol.ClientStrictValidation = false

	return commonConfigure()
}

// chart takes all the received time stamps and put them
// in buckets of 50ms time brackets, it then use the amount
// of messages received in each bucket as the height
func (p *pingCommand) chart() string {
	sort.Float64s(p.times)

	latest := p.times[len(p.times)-1]
	bcount := int(latest/50) + 1
	buckets := make([]float64, bcount)

	for _, t := range p.times {
		b := t / 50.0
		buckets[int(b)]++
	}

	return asciigraph.Plot(
		buckets,
		asciigraph.Height(15),
		asciigraph.Width(60),
		asciigraph.Offset(5),
		asciigraph.Caption("Responses per 50ms"),
	)
}

func init() {
	cli.commands = append(cli.commands, &pingCommand{})
}
