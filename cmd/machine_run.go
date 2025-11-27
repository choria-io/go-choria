// Copyright (c) 2019-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/choria-io/go-choria/aagent"
	aahttp "github.com/choria-io/go-choria/aagent/http"
	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/notifiers/console"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/config"
	"github.com/sirupsen/logrus"
)

type mRunCommand struct {
	command
	sourceDirs []string
	factsFile  string
	dataFile   string
	httpPort   int
	connect    bool
	machines   map[string]*machine.Machine
	haHttp     model.HttpManager
	mu         sync.Mutex
}

func (r *mRunCommand) Setup() (err error) {
	r.machines = make(map[string]*machine.Machine)

	if machine, ok := cmdWithFullCommand("machine"); ok {
		r.cmd = machine.Cmd().Command("run", "Runs an autonomous agent locally")
		r.cmd.Arg("source", "Directories containing the machine definitions").Required().ExistingDirsVar(&r.sourceDirs)
		r.cmd.Flag("facts", "JSON format facts file to supply to the machine as run time facts").ExistingFileVar(&r.factsFile)
		r.cmd.Flag("data", "JSON format data file to supply to the machine as run time data").ExistingFileVar(&r.dataFile)
		r.cmd.Flag("connect", "Connects to the Choria Broker when running the autonomous agent").UnNegatableBoolVar(&r.connect)
		r.cmd.Flag("http", "Starts a HTTP server to interact with the machine").IntVar(&r.httpPort)
	}

	return nil
}

func (r *mRunCommand) Configure() error {
	if r.connect {
		return commonConfigure()
	}

	if debug {
		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Logging at debug level due to CLI override")
	}

	cfg, err = config.NewDefaultConfig()
	if err != nil {
		return err
	}

	cfg.Choria.SecurityProvider = "file"
	cfg.DisableSecurityProviderVerify = true

	return err
}

func (r *mRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if r.httpPort > 0 {
		r.startHttpServer()
	}

	for _, sourceDir := range r.sourceDirs {
		m, err := machine.FromDir(sourceDir, watchers.New(ctx))
		if err != nil {
			return err
		}

		r.mu.Lock()
		r.machines[m.MachineName] = m
		r.mu.Unlock()

		m.SetIdentity("cli")
		m.RegisterNotifier(&console.Notifier{})
		m.SetMainCollective(cfg.MainCollective)
		m.SetExternalMachineNotifier(r.notifyMachinesAfterTransition)
		m.SetExternalMachineStateQuery(r.machineStateLookup)
		if r.haHttp != nil {
			m.SetHttpManager(r.haHttp)
		}

		if r.factsFile != "" {
			facts, err := os.ReadFile(r.factsFile)
			if err != nil {
				return err
			}

			m.SetFactSource(func() json.RawMessage { return facts })
		}

		if r.dataFile != "" {
			dat := make(map[string]any)
			df, err := os.ReadFile(r.dataFile)
			if err != nil {
				return err
			}

			err = json.Unmarshal(df, &dat)
			if err != nil {
				return err
			}

			for k, v := range dat {
				err = m.DataPut(k, v)
				if err != nil {
					return err
				}
			}
		}

		if r.connect {
			conn, err := c.NewConnector(ctx, c.MiddlewareServers, "machine run", c.Logger("machine"))
			if err != nil {
				return err
			}

			m.SetConnection(conn)
		}

		<-m.Start(ctx, wg)
	}

	<-ctx.Done()

	return nil
}

func (r *mRunCommand) notifyMachinesAfterTransition(event *machine.TransitionNotification) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, m := range r.machines {
		logrus.Infof("Notifying %s about event %s from %s", m.MachineName, event.Transition, event.Machine)
		go m.ExternalEventNotify(event)
	}
}

func (r *mRunCommand) machineStateLookup(name string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, m := range r.machines {
		if m.Name() == name {
			return m.State(), nil
		}
	}

	return "", fmt.Errorf("could not find machine matching name='%s'", name)
}

func (r *mRunCommand) startHttpServer() {
	var err error

	r.haHttp, err = aahttp.NewHTTPServer(logrus.WithField("port", r.httpPort))
	if err != nil {
		logrus.Errorf("Could not start HTTP server: %s", err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle(aagent.HTTPSwitchHandlerPattern, aahttp.LoggingMiddleware(logrus.WithField("port", r.httpPort), http.HandlerFunc(r.haHttp.SwitchHandler)))
	mux.Handle(aagent.HTTPMetricHandlerPattern, aahttp.LoggingMiddleware(logrus.WithField("port", r.httpPort), http.HandlerFunc(r.haHttp.MetricHandler)))
	mux.Handle(aagent.HomeAssistantSwitchHandlerPattern, aahttp.LoggingMiddleware(logrus.WithField("port", r.httpPort), http.HandlerFunc(r.haHttp.HASwitchHandler)))

	logrus.Infof("Starting HTTP server on port %d", r.httpPort)

	go func() {
		err = http.ListenAndServe(fmt.Sprintf(":%d", r.httpPort), mux)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}

		logrus.Errorf("HTTP server failed: %s", err)
	}()
}

func init() {
	cli.commands = append(cli.commands, &mRunCommand{})
}
