// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/server/data"
	"github.com/choria-io/go-choria/statistics"
)

// InventoryContent is a fully managed registration plugin for the choria server instance
// it reads the server inventory and publishing it to the collective regularly
type InventoryContent struct {
	c   *config.Config
	log *logrus.Entry
	si  ServerInfoSource
}

const inventoryContentProtocol = "choria:registration:inventorycontent:1"

type InventoryData struct {
	Agents      []agents.Metadata          `json:"agents"`
	Classes     []string                   `json:"classes"`
	Facts       json.RawMessage            `json:"facts"`
	Status      *statistics.InstanceStatus `json:"status"`
	Collectives []string                   `json:"collectives"`
	BuildInfo   *InventoryBuildInfo        `json:"build_info"`
	AutoAgents  []*InventoryMachineState   `json:"machines"`
}

type InventoryBuildInfo struct {
	Version string `json:"version"`
	SHA     string `json:"sha"`
}

type InventoryContentMessage struct {
	Protocol string          `json:"protocol"`
	Content  json.RawMessage `json:"content,omitempty"`
	ZContent []byte          `json:"zcontent,omitempty"`
}

type InventoryMachineState struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	State   string `json:"state"`
}

// NewInventoryContent creates a new fully managed registration plugin instance
func NewInventoryContent(c *config.Config, si ServerInfoSource, logger *logrus.Entry) (*InventoryContent, error) {
	reg := &InventoryContent{si: si}

	reg.Init(c, logger)

	return reg, nil
}

// Init sets up the plugin
func (ic *InventoryContent) Init(c *config.Config, logger *logrus.Entry) {
	ic.c = c
	ic.log = logger.WithFields(logrus.Fields{"registration": "inventory"})

	ic.log.Infof("Configured Inventory Registration")
}

// StartRegistration starts stats a publishing loop
func (ic *InventoryContent) StartRegistration(ctx context.Context, wg *sync.WaitGroup, interval int, output chan *data.RegistrationItem) {
	defer wg.Done()

	delay := time.Duration(rand.N(4)+1) * time.Second
	ic.log.Infof("Sleeping %v before first registration publish", delay)
	err := util.InterruptibleSleep(ctx, delay)
	if err != nil {
		return
	}

	err = ic.publish(output)
	if err != nil {
		ic.log.Errorf("Could not create registration data: %s", err)
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	for {
		select {
		case <-ticker.C:
			err = ic.publish(output)
			if err != nil {
				ic.log.Errorf("Could not create registration data: %s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (ic *InventoryContent) publish(output chan *data.RegistrationItem) error {
	ic.log.Infof("Starting inventory registration poll")

	bi := ic.si.BuildInfo()

	idata := &InventoryData{
		Classes:     ic.si.Classes(),
		Facts:       ic.si.Facts(),
		Collectives: ic.c.Collectives,
		Status:      ic.si.Status(),
		AutoAgents:  []*InventoryMachineState{},
		BuildInfo: &InventoryBuildInfo{
			Version: bi.Version(),
			SHA:     bi.SHA(),
		},
	}

	for _, a := range ic.si.KnownAgents() {
		agent, ok := ic.si.AgentMetadata(a)
		if ok {
			idata.Agents = append(idata.Agents, agent)
		}
	}

	ms, err := ic.si.MachinesStatus()
	if err == nil {
		for _, m := range ms {
			idata.AutoAgents = append(idata.AutoAgents, &InventoryMachineState{
				Name:    m.Name,
				Version: m.Version,
				State:   m.State,
			})
		}
	}

	msg := &InventoryContentMessage{Protocol: inventoryContentProtocol}

	dat, err := json.Marshal(idata)
	if err != nil {
		return err
	}

	if ic.c.Choria.InventoryContentCompression {
		zdat, err := compress(dat)
		if err != nil {
			ic.log.Warnf("Could not compress registration data: %s", err)
		} else {
			msg.ZContent = zdat
		}
	}

	if msg.ZContent == nil {
		msg.Content = dat
	}

	jdat, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("could not json marshal registration message: %s", err)
	}

	item := &data.RegistrationItem{
		Data:        jdat,
		Destination: ic.c.Choria.InventoryContentRegistrationTarget,
	}

	if item.Destination == "" {
		item.TargetAgent = "registration"
	}

	output <- item

	return nil
}
