// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/scoutclient"
	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
)

type ResumeCommand struct {
	fw       inter.Framework
	sopt     *discovery.StandardOptions
	checks   []string
	json     bool
	verbose  bool
	colorize bool
	log      *logrus.Entry
}

func NewResumeCommand(sopt *discovery.StandardOptions, fw inter.Framework, checks []string, json bool, verbose bool, colorize bool, log *logrus.Entry) (*ResumeCommand, error) {
	return &ResumeCommand{
		fw:       fw,
		sopt:     sopt,
		checks:   checks,
		json:     json,
		log:      log,
		verbose:  verbose,
		colorize: colorize,
	}, nil
}

func (t *ResumeCommand) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	sc, err := scoutClient(t.fw, t.sopt, t.log)
	if err != nil {
		return err
	}

	var checks = make([]any, len(t.checks))
	for i, c := range t.checks {
		checks[i] = c
	}

	result, err := sc.Resume().Checks(checks).Do(ctx)
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
	table := iu.NewUTF8TableWithTitle("Scout check resume", "Name", "Triggered", "Skipped", "Failed", "Message")

	result.EachOutput(func(r *scoutclient.ResumeOutput) {
		tr := &scoutagent.ResumeReply{}
		err = r.ParseResumeOutput(tr)
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

		table.AddRow(r.ResultDetails().Sender(), strings.Join(tr.TransitionedChecks, ", "), strings.Join(tr.SkippedChecks, ", "), strings.Join(tr.FailedChecks, ", "), r.ResultDetails().StatusMessage())
	})

	if shown == 0 {
		fmt.Printf("Placed %d checks into normal check mode on %d nodes\n", triggered, result.Stats().ResponsesCount())
		fmt.Println()
	} else {
		fmt.Println(table.Render())
	}

	return result.RenderResults(os.Stdout, scoutclient.TXTFooter, scoutclient.DisplayDDL, t.verbose, false, t.colorize, t.log)
}
