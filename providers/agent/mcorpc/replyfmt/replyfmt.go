// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package replyfmt formats Replies for presentation to users
package replyfmt

import (
	"fmt"
	"io"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/client"
)

// Formatter formats and writes a reply into the bufio writer
type Formatter interface {
	FormatReply(w io.Writer, action ActionDDL, sender string, reply *client.RPCReply) error
	FormatAggregates(w io.Writer, action ActionDDL) error

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

func FormatAggregates(w io.Writer, f OutputFormat, action ActionDDL, opts ...Option) error {
	rf, err := formatter(f, opts...)
	if err != nil {
		return err
	}

	return rf.FormatAggregates(w, action)
}

func FormatReply(w io.Writer, f OutputFormat, action ActionDDL, sender string, reply *client.RPCReply, opts ...Option) error {
	rf, err := formatter(f, opts...)
	if err != nil {
		return err
	}

	return rf.FormatReply(w, action, sender, reply)
}
