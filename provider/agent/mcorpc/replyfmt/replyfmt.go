// Package replyfmt formats Replies for presentation to users
package replyfmt

import (
	"bufio"
	"fmt"

	"github.com/choria-io/mcorpc-agent-provider/mcorpc/client"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
)

// Formatter formats and writes a reply into the bufio writer
type Formatter interface {
	FormatReply(w *bufio.Writer, action *agent.Action, sender string, reply *client.RPCReply) error
	FormatAggregates(w *bufio.Writer, action *agent.Action) error

	SetVerbose()
	SetSilent()
	SetDisplay(mode DisplayMode)
}

// DisplayMode overrides the DDL display hints
type DisplayMode uint8

const (
	DisplayDDL = DisplayMode(iota)
	DisplayOK
	DisplayFailed
	DisplayAll
	DisplayNone
)

// OutputFormat is the format of reply desired
type OutputFormat uint8

const (
	// UnknownFormat is an unknown format
	UnknownFormat = OutputFormat(iota)

	// ConsoleFormat is a format suitable for displaying on the console
	ConsoleFormat
)

// Option configures a formatter
type Option func(f Formatter) error

// verbose sets verbose output mode
func Verbose() Option {
	return func(f Formatter) error {
		f.SetVerbose()

		return nil
	}
}

// silent sets verbose output mode
func Silent() Option {
	return func(f Formatter) error {
		f.SetSilent()

		return nil
	}
}

func Display(d DisplayMode) Option {
	return func(f Formatter) error {
		f.SetDisplay(d)

		return nil
	}
}

func formatter(f OutputFormat, opts ...Option) (Formatter, error) {
	switch f {
	case ConsoleFormat:
		return NewConsoleFormatter(opts...), nil
	default:
		return nil, fmt.Errorf("unknown formatter specified")
	}
}

func FormatAggregates(w *bufio.Writer, f OutputFormat, action *agent.Action, opts ...Option) error {
	rf, err := formatter(f, opts...)
	if err != nil {
		return err
	}

	return rf.FormatAggregates(w, action)
}

func FormatReply(w *bufio.Writer, f OutputFormat, action *agent.Action, sender string, reply *client.RPCReply, opts ...Option) error {
	rf, err := formatter(f, opts...)
	if err != nil {
		return err
	}

	return rf.FormatReply(w, action, sender, reply)
}
