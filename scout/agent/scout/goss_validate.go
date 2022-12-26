// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	gossoutputs "github.com/goss-org/goss/outputs"
)

type GossValidateRequest struct {
	Rules    string `json:"yaml_rules"`
	File     string `json:"file"`
	Vars     string `json:"vars"`
	VarsData string `json:"yaml_vars"`
}

type GossValidateResponse struct {
	Failures int                                `json:"failures"`
	Results  []gossoutputs.StructuredTestResult `json:"results"`
	Runtime  float64                            `json:"runtime"`
	Success  int                                `json:"success"`
	Skipped  int                                `json:"skipped"`
	Summary  string                             `json:"summary"`
	Tests    int                                `json:"tests"`
}
