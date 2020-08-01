package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
	scoutclient "github.com/choria-io/go-choria/scout/client/scout"
)

type StatusCommand struct {
	identity string
	json     bool
	cfile    string
	verbose  bool
	log      *logrus.Entry
}

func NewStatusCommand(id string, jsonf bool, verbose bool, cfile string, log *logrus.Entry) (*StatusCommand, error) {
	return &StatusCommand{
		identity: id,
		json:     jsonf,
		cfile:    cfile,
		log:      log,
		verbose:  verbose,
	}, nil
}

func (s *StatusCommand) Run(ctx context.Context, wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	sc, err := scoutclient.New(scoutclient.ConfigFile(s.cfile), scoutclient.Logger(s.log))
	if err != nil {
		return err
	}

	res, err := sc.OptionTargets([]string{s.identity}).Checks().Do(ctx)
	if err != nil {
		return err
	}

	if s.json {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, s.verbose, false, s.log)
	}

	var outputs []*scoutclient.ChecksOutput
	res.EachOutput(func(o *scoutclient.ChecksOutput) {
		outputs = append(outputs, o)
	})

	if len(outputs) != 1 {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, s.verbose, false, s.log)
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

	return nil
}
