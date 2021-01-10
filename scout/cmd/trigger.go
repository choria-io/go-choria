package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/scoutclient"
	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
)

type TriggerCommand struct {
	sopt     StandardOptions
	checks   []string
	json     bool
	cfile    string
	verbose  bool
	colorize bool
	log      *logrus.Entry
}

func NewTriggerCommand(sopt StandardOptions, checks []string, json bool, cfile string, verbose bool, colorize bool, log *logrus.Entry) (*TriggerCommand, error) {
	return &TriggerCommand{
		sopt:     sopt,
		checks:   checks,
		json:     json,
		log:      log,
		cfile:    cfile,
		verbose:  verbose,
		colorize: colorize,
	}, nil
}

func (t *TriggerCommand) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	sc, err := t.sopt.scoutClient(t.cfile, t.log)
	if err != nil {
		return err
	}

	var checks = make([]interface{}, len(t.checks))
	for i, c := range t.checks {
		checks[i] = c
	}

	result, err := sc.Trigger().Checks(checks).Do(ctx)
	if err != nil {
		return err
	}

	if t.json {
		return result.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, t.verbose, false, t.colorize, t.log)
	}

	if result.Stats().ResponsesCount() == 0 {
		return fmt.Errorf("no responses received")
	}

	mu := sync.Mutex{}
	triggered := 0
	shown := 0
	table := newMarkdownTable("Name", "Triggered", "Skipped", "Failed", "Message")

	result.EachOutput(func(r *scoutclient.TriggerOutput) {
		tr := &scoutagent.TriggerReply{}
		err = r.ParseTriggerOutput(tr)
		if err != nil {
			t.log.Errorf("Could not parse output from %s: %s", r.ResultDetails().Sender(), err)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		triggered += len(tr.TransitionedChecks)

		if !t.verbose && r.ResultDetails().OK() && len(tr.FailedChecks) == 0 {
			return
		}

		shown++

		table.Append([]string{r.ResultDetails().Sender(), strings.Join(tr.TransitionedChecks, ", "), strings.Join(tr.SkippedChecks, ", "), strings.Join(tr.FailedChecks, ", "), r.ResultDetails().StatusMessage()})
	})

	if shown == 0 {
		fmt.Printf("Triggered %d checks on %d nodes\n", triggered, result.Stats().ResponsesCount())
	} else {
		table.Render()
	}

	fmt.Println()
	return result.RenderResults(os.Stdout, scoutclient.TXTFooter, scoutclient.DisplayDDL, t.verbose, false, t.colorize, t.log)
}
