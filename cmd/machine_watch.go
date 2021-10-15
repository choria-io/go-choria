// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers"
)

type mWatchCommand struct {
	command
	onlyTransitions   bool
	onlyWatchers      bool
	filterWatcherType []string
	filterIdentity    string
	filterMachine     string
	log               *logrus.Entry

	sync.Mutex
}

type mWatchableState interface {
	String() string
	SenderID() string
	MachineName() string
}

func (w *mWatchCommand) Setup() (err error) {
	if machine, ok := cmdWithFullCommand("machine"); ok {
		w.cmd = machine.Cmd().Command("watch", "Real time view of machine transitions and watcher states")
		w.cmd.Flag("transitions", "View only transitions").BoolVar(&w.onlyTransitions)
		w.cmd.Flag("watchers", "View only watcher states").BoolVar(&w.onlyWatchers)
		w.cmd.Flag("type", "Limit watcher events to certain types").StringsVar(&w.filterWatcherType)
		w.cmd.Flag("identity", "Filters identity").StringVar(&w.filterIdentity)
		w.cmd.Flag("machine", "Filters based on machine name").StringVar(&w.filterMachine)
	}

	return nil
}

func (w *mWatchCommand) Configure() error {
	return commonConfigure()
}

func (w *mWatchCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	w.log = logrus.NewEntry(c.Logger("x").Logger)

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, c.Certname(), w.log)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	transitions := make(chan inter.ConnectorMessage, 100)
	states := make(chan inter.ConnectorMessage, 100)

	if w.shouldViewTransitions() {
		topic := "choria.machine.transition"
		w.log.Infof("Viewing transitions on topic %s", topic)

		err = conn.QueueSubscribe(ctx, c.UniqueID(), topic, "", transitions)
		if err != nil {
			return fmt.Errorf("could not subscribe to choria.machine.transition: %s", err)
		}
	}

	if w.shouldViewStates() {
		if len(w.filterWatcherType) == 0 {
			w.filterWatcherType = append(w.filterWatcherType, "*")
		}

		for _, ft := range w.filterWatcherType {
			topic := fmt.Sprintf("choria.machine.watcher.%s.state", ft)
			w.log.Infof("Viewing watcher states on topic %s", topic)

			err = conn.QueueSubscribe(ctx, c.UniqueID(), topic, "", states)
			if err != nil {
				return fmt.Errorf("could not subscribe to %s: %s", topic, err)
			}
		}
	}

	var m inter.ConnectorMessage

	for {
		select {
		case m = <-transitions:
			w.showTransition(m)
		case m = <-states:
			w.showState(m)
		case <-ctx.Done():
			return nil
		}
	}
}

func (w *mWatchCommand) dataFromCloudEventJSON(j []byte) ([]byte, error) {
	event := cloudevents.NewEvent("1.0")
	err := event.UnmarshalJSON(j)
	if err != nil {
		return nil, err
	}

	return event.Data(), nil
}

func (w *mWatchCommand) showState(m inter.ConnectorMessage) {
	w.log.Debugf("watcher: topic: %s body: %s", m.Subject(), string(m.Data()))

	data, err := w.dataFromCloudEventJSON(m.Data())
	if err != nil {
		w.log.Errorf("could not parse cloud event: %s", err)
		return
	}

	state, err := watchers.ParseWatcherState(data)
	if err != nil {
		w.log.Errorf("%s", err)
		return
	}

	event, ok := state.(mWatchableState)
	if !ok {
		return
	}

	if w.filterIdentity != "" && !strings.Contains(event.SenderID(), w.filterIdentity) {
		return
	}
	if w.filterMachine != "" && !strings.Contains(event.MachineName(), w.filterMachine) {
		return
	}

	w.Lock()
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), event.String())
	w.Unlock()
}

func (w *mWatchCommand) showTransition(m inter.ConnectorMessage) {
	w.log.Debugf("transition: topic: %s body: %s", m.Subject(), string(m.Data()))

	data, err := w.dataFromCloudEventJSON(m.Data())
	if err != nil {
		w.log.Errorf("could not parse cloud event: %s", err)
		return
	}

	transition := &machine.TransitionNotification{}
	err = json.Unmarshal(data, transition)
	if err != nil {
		w.log.Errorf("Could not decode received transition message: %s: %s", string(data), err)
		return
	}

	if w.filterIdentity != "" && !strings.Contains(transition.Identity, w.filterIdentity) {
		return
	}
	if w.filterMachine != "" && !strings.Contains(transition.Machine, w.filterMachine) {
		return
	}

	w.Lock()
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), transition.String())
	w.Unlock()
}

func (w *mWatchCommand) shouldViewStates() bool {
	return w.onlyWatchers || (!w.onlyTransitions && !w.onlyWatchers)
}

func (w *mWatchCommand) shouldViewTransitions() bool {
	return w.onlyTransitions || (!w.onlyTransitions && !w.onlyWatchers)
}

func init() {
	cli.commands = append(cli.commands, &mWatchCommand{})
}
