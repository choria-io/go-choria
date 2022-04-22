// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package agentharness is a testing framework to mock any RPC agent based on it's DDL
//
// All actions declared in the DDL will be mocked using gomock, expectations can be
// set using the Stub() method
//
// See the harness test integration suite for an example
package agentharness

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo/v2"
	"github.com/sirupsen/logrus"
)

// NewWithDDLBytes creates a new test harness based on a ddl contained in ddlBytes
func NewWithDDLBytes(fw inter.Framework, ctl *gomock.Controller, name string, ddlBytes []byte) (*AgentHarness, error) {
	ddl, err := addl.NewFromBytes(ddlBytes)
	if err != nil {
		return nil, err
	}

	return New(fw, ctl, name, ddl)
}

// NewWithDDLFile creates a new test harness based on a ddl contained on disk
func NewWithDDLFile(fw inter.Framework, ctl *gomock.Controller, name string, ddlFile string) (*AgentHarness, error) {
	ddl, err := addl.New(ddlFile)
	if err != nil {
		return nil, err
	}

	return New(fw, ctl, name, ddl)
}

// New creates a new test harness with all the actions found in ddl mocked using gomock
func New(fw inter.Framework, ctl *gomock.Controller, name string, ddl *addl.DDL) (*AgentHarness, error) {
	h := &AgentHarness{
		name:    name,
		ddl:     ddl,
		fw:      fw,
		log:     fw.Logger(fmt.Sprintf("%s_harnass", name)),
		actions: make(map[string]*MockActionMiddleware),
	}

	if len(ddl.ActionNames()) == 0 {
		return nil, fmt.Errorf("no actions defined")
	}

	for _, action := range ddl.Actions {
		h.actions[action.Name] = NewMockActionMiddleware(ctl)
	}

	return h, nil
}

type ActionMiddleware interface {
	Action(ctx context.Context, req *mcorpc.Request, rep *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo)
}

type AgentHarness struct {
	name    string
	ddl     *addl.DDL
	fw      mcorpc.ChoriaFramework
	log     *logrus.Entry
	actions map[string]*MockActionMiddleware
}

// Stub sets an implementation action to be called for a specific action, panics for
// unknown action
func (h *AgentHarness) Stub(actionName string, action mcorpc.Action) *gomock.Call {
	act, ok := h.actions[actionName]
	if !ok {
		panic(fmt.Sprintf("unknown action %s", actionName))
	}

	return act.EXPECT().Action(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
		func(ctx context.Context, req *mcorpc.Request, rep *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
			action(ctx, req, rep, agent, conn)
		},
	)
}

// Agent creates a valid mcorpc agent ready for passing to a server instance AgentManager
func (h *AgentHarness) Agent() (*mcorpc.Agent, error) {
	agent := mcorpc.New(h.name, h.ddl.Metadata, h.fw, h.log)

	for n := range h.actions {
		act := h.actions[n]
		err := agent.RegisterAction(n, func(ctx context.Context, req *mcorpc.Request, rep *mcorpc.Reply, agent *mcorpc.Agent, conn inter.ConnectorInfo) {
			defer ginkgo.GinkgoRecover()
			act.Action(ctx, req, rep, agent, conn)
		})
		if err != nil {
			return nil, err
		}
	}

	return agent, nil
}
