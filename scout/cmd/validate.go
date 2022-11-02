// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package scoutcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aelsabbahy/goss"
	gossoutputs "github.com/aelsabbahy/goss/outputs"
	"github.com/aelsabbahy/goss/resource"
	gossutil "github.com/aelsabbahy/goss/util"
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
	Display       string
	Table         bool
	Verbose       bool
	Json          bool
	Color         bool
	Local         bool
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

func (v *ValidateCommand) renderTableResult(table *xtablewriter.Table, vr *scoutagent.GossValidateResponse, reqOk bool, sender string, statusMsg string) bool {
	fail := v.fw.Colorize("red", "X")
	ok := v.fw.Colorize("green", "✓")
	skip := v.fw.Colorize("yellow", "?")
	errm := v.fw.Colorize("red", "!")

	should := false

	if !reqOk {
		table.AddRow(fail, sender, "", "", statusMsg)
		return true
	}

	if vr.Failures > 0 || vr.Tests == 0 {
		should = true
		table.AddRow(fail, sender, "", "", vr.Summary)
	} else {
		should = true
		table.AddRow(ok, sender, "", "", vr.Summary)
	}

	sort.Slice(vr.Results, func(i, j int) bool {
		return !vr.Results[i].Successful || vr.Results[i].Err != nil
	})

	if v.opts.Display == "none" {
		return should
	}

	for _, res := range vr.Results {
		should = true

		if res.Err != nil {
			table.AddRow(errm, "", res.ResourceType, res.ResourceId, res.Err.Error())
			continue
		}

		switch {
		case res.Result == resource.SKIP && v.opts.Display != "ok":
			table.AddRow(skip, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: skipped", res.Property))
		case res.Result == resource.SUCCESS && v.opts.Display != "failed":
			for _, e := range res.Expected {
				table.AddRow(ok, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: matches expectation: %v", res.Property, e))
			}
		case res.Result == resource.FAIL && v.opts.Display != "ok":
			for _, e := range res.Expected {
				table.AddRow(fail, "", res.ResourceType, res.ResourceId, fmt.Sprintf("%s: does not match expectation: %v", res.Property, e))
			}
		}
	}

	return should
}

func (v *ValidateCommand) renderTextResult(vr *scoutagent.GossValidateResponse, reqOk bool, sender string, statusMsg string) {
	if !reqOk {
		fmt.Printf("%s: %s\n\n", sender, v.fw.Colorize("red", statusMsg))
		return
	}

	if vr.Failures > 0 || vr.Tests == 0 {
		fmt.Printf("%s: %s\n\n", sender, v.fw.Colorize("red", vr.Summary))
	} else {
		fmt.Printf("%s: %s\n\n", sender, v.fw.Colorize("green", vr.Summary))
	}

	sort.Slice(vr.Results, func(i, j int) bool {
		return !vr.Results[i].Successful
	})

	if v.opts.Display == "none" {
		fmt.Println()
		return
	}

	lb := false
	for i, res := range vr.Results {
		switch {
		case res.Result == resource.SKIP && v.opts.Display != "ok":
			if lb {
				fmt.Println()
			}
			fmt.Printf("   %s %s: %s: %s: skipped\n", v.fw.Colorize("yellow", "?"), res.ResourceType, res.ResourceId, res.Property)
			lb = false
		case res.Result == resource.FAIL && v.opts.Display != "ok":
			if i != 0 {
				fmt.Println()
			}
			lb = true
			msg := fmt.Sprintf("%s %s", v.fw.Colorize("red", "X"), res.SummaryLine)
			fmt.Printf("%s\n", iu.ParagraphPadding(msg, 3))
		case res.Result == resource.SUCCESS && v.opts.Display != "failed":
			if lb {
				fmt.Println()
			}
			for _, e := range res.Expected {
				fmt.Printf("   %s %s: %s: %s: matches expectation: %v\n", v.fw.Colorize("green", "✓"), res.ResourceType, res.ResourceId, res.Property, e)
			}

			lb = false
		}
	}

	fmt.Println()
}

func (v *ValidateCommand) localValidate() error {
	var err error
	var out bytes.Buffer
	var table *xtablewriter.Table
	var shouldRenderTable bool

	rules, err := os.CreateTemp("", "choria-gossfile-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(rules.Name())
	defer rules.Close()

	_, err = rules.Write(v.opts.Rules)
	if err != nil {
		return err
	}
	rules.Close()

	opts := []gossutil.ConfigOption{
		gossutil.WithMaxConcurrency(1),
		gossutil.WithResultWriter(&out),
		gossutil.WithSpecFile(rules.Name()),
	}

	if len(v.opts.Variables) > 0 {
		opts = append(opts, gossutil.WithVarsBytes(v.opts.Variables))
	}

	cfg, err := gossutil.NewConfig(opts...)
	if err != nil {
		return err
	}

	_, err = goss.Validate(cfg, time.Now())
	if err != nil {
		return err
	}

	res := &gossoutputs.StructuredOutput{}
	err = json.Unmarshal(out.Bytes(), res)
	if err != nil {
		return err
	}

	resp := &scoutagent.GossValidateResponse{Results: []gossoutputs.StructuredTestResult{}}

	var errors int
	for _, r := range res.Results {
		switch {
		case r.Err != nil:
			errors++
		case r.Result == resource.SKIP:
			resp.Skipped++
		}
	}

	resp.Results = res.Results
	resp.Summary = res.SummaryLine
	resp.Failures = res.Summary.Failed + errors
	resp.Runtime = res.Summary.TotalDuration.Seconds()
	resp.Success = res.Summary.TestCount - res.Summary.Failed - resp.Skipped
	resp.Tests = res.Summary.TestCount

	if v.opts.Table {
		table = iu.NewUTF8TableWithTitle("Goss check results", "", "Node", "Resource", "ID", "State")
	}

	if v.opts.Table {
		shouldRenderTable = v.renderTableResult(table, resp, true, "localhost", "OK")
	} else {
		v.renderTextResult(resp, true, "localhost", "OK")
	}

	if v.opts.Table && shouldRenderTable {
		fmt.Println(table.Render())
	}

	return nil
}

func (v *ValidateCommand) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	if v.opts.NodeRulesFile == "" && len(v.opts.Rules) == 0 {
		return fmt.Errorf("neither local validation rules nor a remote file were supplied")
	}
	if v.opts.NodeRulesFile != "" && len(v.opts.Rules) > 0 {
		return fmt.Errorf("both local validation rules and a remote rules file were supplied")
	}
	if len(v.opts.Variables) > 0 && v.opts.NodeVarsFile != "" {
		return fmt.Errorf("both local variables and a remote variables file were supplied")
	}

	if v.opts.Local {
		return v.localValidate()
	}

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
			shouldRenderTable = v.renderTableResult(table, vr, r.ResultDetails().OK(), r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
		} else {
			v.renderTextResult(vr, r.ResultDetails().OK(), r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
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
