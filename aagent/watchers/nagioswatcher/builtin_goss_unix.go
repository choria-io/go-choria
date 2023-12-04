// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package nagioswatcher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/goss-org/goss"
	"github.com/goss-org/goss/outputs"
	gossutil "github.com/goss-org/goss/util"
)

func (w *Watcher) watchUsingGoss() (state State, output string, err error) {
	var out bytes.Buffer

	opts := []gossutil.ConfigOption{
		gossutil.WithMaxConcurrency(1),
		gossutil.WithResultWriter(&out),
		gossutil.WithSpecFile(w.properties.Gossfile),
	}

	od, err := w.machine.OverrideData()
	if err == nil && len(od) > 0 {
		opts = append(opts, gossutil.WithVarsBytes(od))
	}

	cfg, err := gossutil.NewConfig(opts...)
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss configuration failed: %s", err), err
	}

	_, err = goss.Validate(cfg)
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss validate failed: %s", err), err
	}

	res := &outputs.StructuredOutput{}
	err = json.Unmarshal(out.Bytes(), res)
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss output invalid: %s", err), err
	}

	pd := fmt.Sprintf("checks=%d;; failed=%d;; runtime=%fs", res.Summary.TestCount, res.Summary.Failed, res.Summary.TotalDuration.Seconds())

	if res.Summary.Failed > 0 {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", res.SummaryLine, pd), nil
	}

	return OK, fmt.Sprintf("OK: %s|%s", res.SummaryLine, pd), nil
}
