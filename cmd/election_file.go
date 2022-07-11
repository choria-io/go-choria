// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	election "github.com/choria-io/go-choria/providers/election/streams"
	"github.com/sirupsen/logrus"
)

type tElectFileCommand struct {
	command
	file   string
	name   string
	bucket string
	cnt    int
	log    *logrus.Entry
	mu     sync.Mutex
}

func (f *tElectFileCommand) Setup() (err error) {
	if elect, ok := cmdWithFullCommand("election"); ok {
		f.cmd = elect.Cmd().Command("file", "Maintains a file based on Leader Elections")
		f.cmd.Arg("name", "The name for the Leader Election to campaign in").Required().StringVar(&f.name)
		f.cmd.Arg("file", "The file to maintain under election").Required().StringVar(&f.file)
		f.cmd.Flag("bucket", "Use a specific bucket for elections").Default("CHORIA_LEADER_ELECTION").StringVar(&f.bucket)
	}

	return nil
}

func (f *tElectFileCommand) Configure() (err error) {
	return commonConfigure()
}

func (f *tElectFileCommand) remove() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if choria.FileExist(f.file) {
		err := os.Remove(f.file)
		if err != nil {
			f.log.Errorf("Could not remove managed file %s: %v", f.file, err)
		}
	}

	f.cnt = 0
}

func (f *tElectFileCommand) create() {
	f.mu.Lock()
	defer f.mu.Unlock()

	data := map[string]interface{}{
		"timestamp": time.Now(),
		"count":     f.cnt,
	}
	dj, _ := json.Marshal(data)
	err = os.WriteFile(f.file, dj, 0644)
	if err != nil {
		f.log.Errorf("Could not write managed file %s: %v", f.file, err)
	}
	f.cnt++
}

func (f *tElectFileCommand) won() {
	f.log.Infof("Became leader")
	f.create()
}

func (f *tElectFileCommand) lost() {
	f.log.Infof("Lost ladership")
	f.remove()
}

func (f *tElectFileCommand) campaign(s election.State) {
	switch s {
	case election.LeaderState:
		f.create()

	default:
		f.remove()
	}
}

func (f *tElectFileCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	f.log = c.Logger("election")

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("election %s %s", f.name, c.Config.Identity), f.log)
	if err != nil {
		return err
	}

	js, err := conn.Nats().JetStream()
	if err != nil {
		return err
	}

	kv, err := js.KeyValue(f.bucket)
	if err != nil {
		return fmt.Errorf("cannot access KV Bucket %s: %v", f.bucket, err)
	}

	el, err := election.NewElection(c.Config.Identity, f.name, kv, election.OnWon(f.won), election.OnLost(f.lost), election.OnCampaign(f.campaign))
	if err != nil {
		return err
	}

	return el.Start(ctx)
}

func init() {
	cli.commands = append(cli.commands, &tElectFileCommand{})
}
