package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	scoutcmd "github.com/choria-io/go-choria/scout/cmd"
)

type sWatchCommand struct {
	identity string
	check    string
	perf     bool
	history  time.Duration
	log      *logrus.Entry
	watch    *scoutcmd.WatchCommand

	command
	sync.Mutex
}

func (w *sWatchCommand) Setup() (err error) {
	if scout, ok := cmdWithFullCommand("scout"); ok {
		w.cmd = scout.Cmd().Command("watch", "Watch CloudEvents produced by Scout")
		w.cmd.Flag("identity", "Filters events by identity").StringVar(&w.identity)
		w.cmd.Flag("check", "Filters events by check").StringVar(&w.check)
		w.cmd.Flag("perf", "Show performance data").BoolVar(&w.perf)
		w.cmd.Flag("history", "Retrieve a certain period of history from Choria Streaming Server").DurationVar(&w.history)
	}

	return nil
}

func (w *sWatchCommand) Configure() error {
	return commonConfigure()
}

func (w *sWatchCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	w.log = logrus.NewEntry(c.Logger("scout").Logger)

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, c.Certname(), w.log)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}

	w.watch, err = scoutcmd.NewWatchCommand(w.identity, w.check, w.perf, w.history, conn, w.log)
	if err != nil {
		return err
	}

	wg.Add(1)
	return w.watch.Run(ctx, wg)
}

func init() {
	cli.commands = append(cli.commands, &sWatchCommand{})
}
