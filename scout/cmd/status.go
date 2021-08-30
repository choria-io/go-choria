package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/scoutclient"
	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
)

type StatusCommand struct {
	identity string
	json     bool
	fw       inter.Framework
	verbose  bool
	colorize bool
	log      *logrus.Entry
}

func NewStatusCommand(fw inter.Framework, id string, jsonf bool, verbose bool, colorize bool, log *logrus.Entry) (*StatusCommand, error) {
	return &StatusCommand{
		identity: id,
		json:     jsonf,
		fw:       fw,
		log:      log,
		verbose:  verbose,
		colorize: colorize,
	}, nil
}

func (s *StatusCommand) Run(ctx context.Context, wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	sc, err := scoutclient.New(s.fw, scoutclient.Logger(s.log), scoutclient.Progress())
	if err != nil {
		return err
	}

	res, err := sc.OptionTargets([]string{s.identity}).Checks().Do(ctx)
	if err != nil {
		return err
	}

	if s.json {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, s.verbose, false, s.colorize, s.log)
	}

	var outputs []*scoutclient.ChecksOutput
	res.EachOutput(func(o *scoutclient.ChecksOutput) {
		outputs = append(outputs, o)
	})

	if len(outputs) != 1 {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, s.verbose, false, s.colorize, s.log)
	}

	if !outputs[0].ResultDetails().OK() {
		return fmt.Errorf("loading checks failed: %s", outputs[0].ResultDetails().StatusMessage())
	}

	checks := scoutagent.ChecksResponse{}
	err = outputs[0].ParseChecksOutput(&checks)
	if err != nil {
		return err
	}

	table := newMarkdownTable("Name", "State", "Last Check", "History")

	for _, c := range checks.Checks {
		last := "Never"
		history := []string{}
		if c.Status != nil {
			if c.Status.CheckTime != 0 {
				last = time.Since(time.Unix(c.Status.CheckTime, 0)).Round(time.Second).String()
			}

			hist := c.Status.History
			if len(hist) > 10 {
				hist = hist[len(hist)-10:]
			}

			for _, h := range hist {
				switch h.Status {
				case 0:
					history = append(history, "OK")
				case 1:
					history = append(history, "WA")
				case 2:
					history = append(history, "CR")
				default:
					history = append(history, "UN")
				}
			}
		}

		table.Append([]string{c.Name, c.State, last, strings.Join(history, " ")})
	}

	table.Render()

	fmt.Println()
	return res.RenderResults(os.Stdout, scoutclient.TXTFooter, scoutclient.DisplayDDL, s.verbose, false, s.colorize, s.log)
}
