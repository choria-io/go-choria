// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"
)

// RPCReply is a basic RPC reply
type RPCReply struct {
	Action     string            `json:"action"`
	Statuscode mcorpc.StatusCode `json:"statuscode"`
	Statusmsg  string            `json:"statusmsg"`
	Data       json.RawMessage   `json:"data"`
	Sender     string            `json:"sender"`
	Time       time.Time         `json:"time_utc"`
}

// MatchExpr determines if the Reply  matches expression q using the expr format.
// The query q is expected to return a boolean type else an error will be raised
func (r *RPCReply) MatchExpr(q string, prog *vm.Program) (bool, *vm.Program, error) {
	env := map[string]any{
		"msg":            r.Statusmsg,
		"code":           int(r.Statuscode),
		"data":           r.lookup,
		"ok":             r.isOK,
		"aborted":        r.isAborted,
		"invalid_data":   r.isInvalidData,
		"missing_data":   r.isMissingData,
		"unknown_action": r.isUnknownAction,
		"unknown_error":  r.isUnknownError,
		"include":        r.include,
		"semver":         r.semverCompare,
		"sender":         func() string { return r.Sender },
		"time":           func() time.Time { return r.Time },
	}

	var err error
	if prog == nil {
		prog, err = expr.Compile(q, expr.AllowUndefinedVariables(), expr.Env(env))
		if err != nil {
			return false, nil, err
		}
	}

	out, err := expr.Run(prog, env)
	if err != nil {
		return false, prog, err
	}

	matched, ok := out.(bool)
	if !ok {
		return false, prog, fmt.Errorf("match expressions should return boolean")
	}

	return matched, prog, nil
}

func (r *RPCReply) isOK() bool {
	return r.Statuscode == mcorpc.OK
}

func (r *RPCReply) isAborted() bool {
	return r.Statuscode == mcorpc.Aborted
}

func (r *RPCReply) isUnknownAction() bool {
	return r.Statuscode == mcorpc.UnknownAction
}

func (r *RPCReply) isMissingData() bool {
	return r.Statuscode == mcorpc.MissingData
}

func (r *RPCReply) isInvalidData() bool {
	return r.Statuscode == mcorpc.InvalidData
}

func (r *RPCReply) isUnknownError() bool {
	return r.Statuscode == mcorpc.UnknownError
}

// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
func (r *RPCReply) lookup(query string) any {
	return gjson.GetBytes(r.Data, query).Value()
}

func (r *RPCReply) semverCompare(value string, cmp string) (bool, error) {
	cons, err := semver.NewConstraint(cmp)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(value)
	if err != nil {
		return false, err
	}

	return cons.Check(v), nil
}

func (r *RPCReply) include(hay []any, needle any) bool {
	// gjson always turns numbers into float64
	i, ok := needle.(int)
	if ok {
		needle = float64(i)
	}

	for _, i := range hay {
		if cmp.Equal(i, needle) {
			return true
		}
	}

	return false
}
