package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
	scoutclient "github.com/choria-io/go-choria/scout/client/scout"
)

type ChecksCommand struct {
	identity string
	json     bool
	cfile    string
	verbose  bool
	log      *logrus.Entry
}

func NewChecksCommand(id string, jsonf bool, verbose bool, cfile string, log *logrus.Entry) (*ChecksCommand, error) {
	return &ChecksCommand{identity: id, json: jsonf, cfile: cfile, log: log, verbose: verbose}, nil
}

func (w *ChecksCommand) Run(ctx context.Context, wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	sc, err := scoutclient.New(scoutclient.ConfigFile(w.cfile), scoutclient.Logger(w.log))
	if err != nil {
		return err
	}

	res, err := sc.OptionTargets([]string{w.identity}).Checks().Do(ctx)
	if err != nil {
		return err
	}

	if w.json {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, w.verbose, false, w.log)
	}

	var outputs []*scoutclient.ChecksOutput
	res.EachOutput(func(o *scoutclient.ChecksOutput) {
		outputs = append(outputs, o)
	})

	if len(outputs) != 1 {
		return res.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, w.verbose, false, w.log)
	}

	if !outputs[0].ResultDetails().OK() {
		return fmt.Errorf("loading checks failed: %s", outputs[0].ResultDetails().StatusMessage())
	}

	checks := scoutagent.ChecksResponse{}
	err = outputs[0].ParseChecksOutput(&checks)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Name", "State", "Last Check", "History"})

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
