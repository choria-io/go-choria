package cmd

import (
	"sync"

	"github.com/choria-io/go-choria/events"
)

type eventCommand struct {
	command

	componentF string
	typeF      string
}

// tool event
func (e *eventCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		e.cmd = tool.Cmd().Command("event", "View Choria lifecycle events")
		e.cmd.Flag("component", "Limit events to a named component").StringVar(&e.componentF)
		e.cmd.Flag("type", "Limits the events to a particular type").EnumVar(&e.typeF, events.EventTypeNames()...)
	}

	return nil
}

func (e *eventCommand) Configure() error {
	return commonConfigure()
}

func (e *eventCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	opt := &events.ViewOptions{
		Choria:          c,
		ComponentFilter: e.componentF,
		TypeFilter:      e.typeF,
		Debug:           debug,
	}

	events.View(ctx, opt)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &eventCommand{})
}
