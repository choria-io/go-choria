package cmd

import (
	"sync"

	lifecycle "github.com/choria-io/go-lifecycle"
)

type eventCommand struct {
	command

	componentF string
	typeF      string
}

func (e *eventCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		e.cmd = tool.Cmd().Command("event", "View Choria lifecycle events")
		e.cmd.Flag("component", "Limit events to a named component").StringVar(&e.componentF)
		e.cmd.Flag("type", "Limits the events to a particular type").EnumVar(&e.typeF, lifecycle.EventTypeNames()...)
	}

	return nil
}

func (e *eventCommand) Configure() error {
	return commonConfigure()
}

func (e *eventCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	opt := &lifecycle.ViewOptions{
		Choria:          c,
		ComponentFilter: e.componentF,
		TypeFilter:      e.typeF,
		Debug:           debug,
	}

	lifecycle.View(ctx, opt)

	return nil
}

func init() {
	cli.commands = append(cli.commands, &eventCommand{})
}
