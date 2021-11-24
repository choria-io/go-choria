// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/guptarohit/asciigraph"
	log "github.com/sirupsen/logrus"
)

type pingCommand struct {
	command

	silent  bool
	timeout int
	graph   bool
	workers int
	waitfor int

	fo *discovery.StandardOptions

	namesOnly bool

	start     time.Time
	published time.Time

	ctr int
	mu  *sync.Mutex

	times []time.Duration
}

func (p *pingCommand) Setup() (err error) {
	p.cmd = cli.app.Command("ping", "Low level Choria network testing tool")
	p.cmd.Flag("config", "Config file to use").PlaceHolder("FILE").StringVar(&configFile)
	p.cmd.Flag("silent", "Do not print any hostnames").BoolVar(&p.silent)
	p.cmd.Flag("names", "Only show the names that respond, no statistics").BoolVar(&p.namesOnly)

	p.fo = discovery.NewStandardOptions()
	p.fo.AddFilterFlags(p.cmd)

	p.cmd.Flag("expect", "Wait until this many replies were received or timeout").IntVar(&p.waitfor)
	p.cmd.Flag("timeout", "How long to wait for responses").IntVar(&p.timeout)
	p.cmd.Flag("graph", "Produce a graph of the result times").BoolVar(&p.graph)
	p.cmd.Flag("workers", "How many workers to start for receiving messages").Default("3").IntVar(&p.workers)

	p.start = time.Now()

	return
}

func (p *pingCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	p.times = []time.Duration{}
	p.mu = &sync.Mutex{}

	if p.timeout == 0 {
		p.timeout = cfg.DiscoveryTimeout
	}

	p.fo.SetDefaultsFromChoria(c)

	filter, err := p.fo.NewFilter("")

	if err != nil {
		return fmt.Errorf("could not parse filters: %s", err)
	}

	msg, err := p.createMessage(protocol.NewFilter())
	if err != nil {
		return fmt.Errorf("could not create message: %s", err)
	}
	msg.SetFilter(filter)
	msg.OnPublish(func() {
		if p.published.IsZero() {
			p.published = time.Now()
		}
	})

	cl, err := client.New(c, client.Receivers(p.workers), client.Timeout(time.Duration(p.timeout)*time.Second))
	if err != nil {
		return fmt.Errorf("could not setup client: %s", err)
	}

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
		sum := time.Duration(0)
		min := time.Duration(math.MaxInt64)
		max := time.Duration(0)
		avg := time.Duration(0)

		for _, value := range p.times {
			sum += value
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}

		avg = sum / time.Duration(len(p.times))

		fmt.Printf("%d replies max: %s min: %s avg: %s overhead: %s\n", len(p.times), max.Round(time.Millisecond), min.Round(time.Millisecond), avg.Round(time.Millisecond), p.published.Sub(p.start).Round(time.Millisecond))

		if p.graph {
			fmt.Println()
			fmt.Println(p.chart())
			fmt.Println()
		}

		return nil
	}

	return errors.New("no responses received")
}

func (p *pingCommand) handler(_ context.Context, m inter.ConnectorMessage) {
	reply, err := c.NewTransportFromJSON(string(m.Data()))
	if err != nil {
		log.Errorf("Could not process a reply: %s", err)
		return
	}

	now := time.Now()
	delay := now.Sub(p.published)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.times = append(p.times, delay)

	if !p.silent {
		if p.namesOnly {
			fmt.Println(reply.SenderID())
		} else {
			fmt.Printf("%-40s time=%d ms\n", reply.SenderID(), delay.Milliseconds())
		}
	}

	p.ctr++
	if p.waitfor == p.ctr {
		cancel()
	}
}

func (p *pingCommand) createMessage(filter *protocol.Filter) (inter.Message, error) {
	msg, err := c.NewMessage(base64.StdEncoding.EncodeToString([]byte("ping")), "discovery", p.fo.Collective, "request", nil)
	if err != nil {
		return nil, fmt.Errorf("could not create message: %s", err)
	}

	msg.SetProtocolVersion(protocol.RequestV1)
	msg.SetReplyTo(choria.ReplyTarget(msg, msg.RequestID()))

	msg.SetFilter(filter)

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
	sort.Slice(p.times, func(i, j int) bool {
		return p.times[i] < p.times[j]
	})

	ftimes := make([]int64, len(p.times))
	for i, d := range p.times {
		ftimes[i] = d.Milliseconds()
	}

	latest := ftimes[len(ftimes)-1]
	bcount := int(latest/50) + 1
	buckets := make([]float64, bcount)

	for _, t := range ftimes {
		buckets[t/50]++
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
