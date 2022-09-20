// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aelsabbahy/goss"
	gossoutputs "github.com/aelsabbahy/goss/outputs"
	"github.com/aelsabbahy/goss/resource"
	gossutil "github.com/aelsabbahy/goss/util"
	"github.com/choria-io/go-choria/inter"

	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

func gossValidateAction(_ context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, _ inter.ConnectorInfo) {
	resp := &GossValidateResponse{Results: []gossoutputs.StructuredTestResult{}}
	reply.Data = resp

	args := &GossValidateRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	switch {
	case args.Rules == "" && args.File == "":
		abort("One of rules or file is required", reply)
		return
	case args.Rules != "" && args.File != "":
		abort("Only one of rules or file can be supplied", reply)
		return
	case args.Rules != "":
		tf, err := os.CreateTemp("", fmt.Sprintf("choria-gossfile-%s-*.yaml", req.RequestID))
		if err != nil {
			agent.Log.Errorf("Writing gossfile failed: %v", err)
			abort("Could not create gossfile", reply)
			return
		}
		defer os.Remove(tf.Name())
		tf.Close()

		err = os.WriteFile(tf.Name(), []byte(args.Rules), 0400)
		if err != nil {
			agent.Log.Errorf("Writing gossfile failed: %v", err)
			abort("Could not create gossfile", reply)
			return
		}
		args.File = tf.Name()
	}

	switch {
	case args.VarsData != "" && args.Vars != "":
		abort("Only one of yaml_vars or vars can be supplied", reply)
		return
	case args.VarsData != "":
		tf, err := os.CreateTemp("", fmt.Sprintf("choria-gossvars-%s-*.yaml", req.RequestID))
		if err != nil {
			agent.Log.Errorf("Writing goss variables file failed: %v", err)
			abort("Could not create variables file", reply)
			return
		}
		defer os.Remove(tf.Name())
		tf.Close()

		err = os.WriteFile(tf.Name(), []byte(args.VarsData), 0400)
		if err != nil {
			agent.Log.Errorf("Writing variables file failed: %v", err)
			abort("Could not create variables file", reply)
			return
		}
		args.Vars = tf.Name()
	}

	var out bytes.Buffer

	opts := []gossutil.ConfigOption{
		gossutil.WithMaxConcurrency(1),
		gossutil.WithResultWriter(&out),
		gossutil.WithSpecFile(args.File),
	}

	if args.Vars != "" {
		opts = append(opts, gossutil.WithVarsFile(args.Vars))
	}

	cfg, err := gossutil.NewConfig(opts...)
	if err != nil {
		abort(fmt.Sprintf("Could not create Goss config: %s", err), reply)
		return
	}

	_, err = goss.Validate(cfg, time.Now())
	if err != nil {
		abort(fmt.Sprintf("Could not validate: %s", err), reply)
		return
	}

	res := &gossoutputs.StructuredOutput{}
	err = json.Unmarshal(out.Bytes(), res)
	if err != nil {
		abort(fmt.Sprintf("Could not parse goss results: %s", err), reply)
		return
	}

	for _, r := range res.Results {
		if r.Result == resource.SKIP {
			resp.Skipped++
		}
	}
	resp.Results = res.Results
	resp.Summary = res.SummaryLine
	resp.Failures = res.Summary.Failed
	resp.Runtime = res.Summary.TotalDuration.Seconds()
	resp.Success = res.Summary.TestCount - res.Summary.Failed - resp.Skipped
	resp.Tests = res.Summary.TestCount

}
