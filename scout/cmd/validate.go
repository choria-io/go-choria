// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package scoutcmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aelsabbahy/goss/resource"
	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/scoutclient"
	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	scoutagent "github.com/choria-io/go-choria/scout/agent/scout"
	"github.com/sirupsen/logrus"
	xtablewriter "github.com/xlab/tablewriter"
)

type ValidateCommandOptions struct {
	Variables     []byte
	NodeVarsFile  string
	Rules         []byte
	NodeRulesFile string
	KVRules       string
	KVVariables   string
	Display       string
	Table         bool
	Verbose       bool
	Json          bool
	Color         bool
}

type ValidateCommand struct {
	sopts *discovery.StandardOptions
	log   *logrus.Entry
	fw    inter.Framework
	opts  *ValidateCommandOptions
}

func NewValidateCommand(sopts *discovery.StandardOptions, fw inter.Framework, opts *ValidateCommandOptions, log *logrus.Entry) (*ValidateCommand, error) {
	return &ValidateCommand{
		sopts: sopts,
		log:   log,
		fw:    fw,
		opts:  opts,
	}, nil
}

func (v *ValidateCommand) renderTableResult(table *xtablewriter.Table, vr *scoutagent.GossValidateResponse, r *scoutclient.GossValidateOutput) bool {
	fail := v.fw.Colorize("red", "X")
	ok := v.fw.Colorize("green", "✓")
	skip := v.fw.Colorize("yellow", "?")

	should := false

	if !r.ResultDetails().OK() {
		table.AddRow(fail, r.ResultDetails().Sender(), "", "", r.ResultDetails().StatusMessage())
		return true
	}

	if vr.Failures > 0 || vr.Tests == 0 {
		should = true
		table.AddRow(fail, r.ResultDetails().Sender(), "", "", vr.Summary)
	} else {
		should = true
		table.AddRow(ok, r.ResultDetails().Sender(), "", "", vr.Summary)
	}

	sort.Slice(vr.Results, func(i, j int) bool {
		return !vr.Results[i].Successful
	})

	for _, res := range vr.Results {
		should = true
		switch res.Result {
		case resource.SKIP:
			table.AddRow(skip, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: skipped", res.Property))
		case resource.SUCCESS:
			table.AddRow(ok, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: matches expectation: %v", res.Property, res.Expected))
		case resource.FAIL:
			table.AddRow(fail, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: does not match expectation: %v", res.Property, res.Expected))
		}
	}

	return should
}

func (v *ValidateCommand) renderTextResult(vr *scoutagent.GossValidateResponse, r *scoutclient.GossValidateOutput) {
	if !r.ResultDetails().OK() {
		fmt.Printf("%s: %s\n\n", r.ResultDetails().Sender(), v.fw.Colorize("red", r.ResultDetails().StatusMessage()))
		return
	}

	if vr.Failures > 0 || vr.Tests == 0 {
		fmt.Printf("%s: %s\n\n", r.ResultDetails().Sender(), v.fw.Colorize("red", vr.Summary))
	} else {
		fmt.Printf("%s: %s\n\n", r.ResultDetails().Sender(), v.fw.Colorize("green", vr.Summary))
	}

	sort.Slice(vr.Results, func(i, j int) bool {
		return !vr.Results[i].Successful
	})

	lb := false
	for i, res := range vr.Results {
		switch res.Result {
		case resource.SKIP:
			if lb {
				fmt.Println()
			}
			fmt.Printf("   %s %s: %s: %s: skipped\n", v.fw.Colorize("yellow", "?"), res.ResourceType, res.ResourceId, res.Property)
			lb = false
		case resource.FAIL:
			if i != 0 {
				fmt.Println()
			}
			lb = true
			msg := fmt.Sprintf("%s %s", v.fw.Colorize("red", "X"), res.SummaryLine)
			fmt.Printf("%s\n", iu.ParagraphPadding(msg, 3))
		case resource.SUCCESS:
			if lb {
				fmt.Println()
			}
			fmt.Printf("   %s %s: %s: %s: matches expectation: %v\n", v.fw.Colorize("green", "✓"), res.ResourceType, res.ResourceId, res.Property, res.Expected)
			lb = false
		}
	}
	fmt.Println()
}

func (v *ValidateCommand) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	sc, err := scoutClient(v.fw, v.sopts, v.log)
	if err != nil {
		return err
	}

	action := sc.GossValidate()
	if v.opts.NodeRulesFile != "" {
		action.File(v.opts.NodeRulesFile)
	} else if len(v.opts.Rules) > 0 {
		action.YamlRules(string(v.opts.Rules))
	} else {
		return fmt.Errorf("no rules or rules file specified")
	}

	if len(v.opts.Variables) > 0 {
		action.YamlVars(string(v.opts.Variables))
	} else if v.opts.NodeVarsFile != "" {
		action.Vars(v.opts.NodeVarsFile)
	}

	start := time.Now()
	result, err := action.Do(ctx)
	if err != nil {
		return err
	}
	runTime := time.Since(start)

	if v.opts.Json {
		return result.RenderResults(os.Stdout, scoutclient.JSONFormat, scoutclient.DisplayDDL, v.opts.Verbose, false, v.opts.Color, v.log)
	}

	if result.Stats().ResponsesCount() == 0 {
		return fmt.Errorf("no responses received")
	}

	count := 0
	failed := 0
	success := 0
	skipped := 0
	nodes := 0
	shouldRenderTable := false

	var table *xtablewriter.Table
	if v.opts.Table {
		table = iu.NewUTF8TableWithTitle("Goss check results", "", "Node", "Resource", "ID", "State")
	}

	result.EachOutput(func(r *scoutclient.GossValidateOutput) {
		vr := &scoutagent.GossValidateResponse{}
		err = r.ParseGossValidateOutput(vr)
		if err != nil {
			v.log.Errorf("Could not parse output from %s: %s", r.ResultDetails().Sender(), err)
			return
		}

		nodes++
		count += vr.Tests
		failed += vr.Failures
		success += vr.Success
		skipped += vr.Skipped
		if !r.ResultDetails().OK() {
			failed++
		}

		switch v.opts.Display {
		case "none":
			return
		case "all":
		case "ok":
			// skip on not ok
			if !r.ResultDetails().OK() || vr.Tests == 0 || vr.Failures > 0 || vr.Skipped > 0 {
				return
			}
		case "failed":
			// skip all ok
			if r.ResultDetails().OK() && vr.Tests > 0 && vr.Failures == 0 && vr.Skipped == 0 {
				return
			}
		}

		if v.opts.Table {
			shouldRenderTable = v.renderTableResult(table, vr, r)
		} else {
			v.renderTextResult(vr, r)
		}
	})

	if v.opts.Table && shouldRenderTable {
		fmt.Println(table.Render())
	}

	parts := []string{
		fmt.Sprintf("Nodes: %d", nodes),
	}
	if failed > 0 {
		parts = append(parts, v.fw.Colorize("red", fmt.Sprintf("Failed: %d", failed)))
	} else {
		parts = append(parts, v.fw.Colorize("green", fmt.Sprintf("Failed: %d", failed)))
	}
	if skipped > 0 {
		parts = append(parts, v.fw.Colorize("yellow", fmt.Sprintf("Skipped: %d", skipped)))
	} else {
		parts = append(parts, v.fw.Colorize("green", fmt.Sprintf("Skipped: %d", skipped)))
	}
	if success > 0 {
		parts = append(parts, v.fw.Colorize("green", fmt.Sprintf("Success: %d", success)))
	} else {
		parts = append(parts, v.fw.Colorize("red", fmt.Sprintf("Success: %d", success)))
	}
	parts = append(parts, fmt.Sprintf("Duration: %v", runTime.Round(time.Millisecond)))

	fmt.Printf("%s\n", strings.Join(parts, ", "))

	if v.opts.Verbose {
		return result.RenderResults(os.Stdout, scoutclient.TXTFooter, scoutclient.DisplayDDL, v.opts.Verbose, false, v.opts.Color, v.log)
	}

	return nil
}
