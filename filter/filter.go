// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/protocol"
)

// Filter is a function that represents a specific filter in string form
type Filter func(f *protocol.Filter) error

// AgentFilter processes a series of strings as agent filters
func AgentFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			if filter == "" {
				continue
			}

			pf.AddAgentFilter(filter)
		}

		return
	}
}

// ClassFilter processes a series of strings as agent filters
func ClassFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			if filter == "" {
				continue
			}

			pf.AddClassFilter(filter)
		}

		return
	}
}

// IdentityFilter processes a series of strings as identity filters
func IdentityFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			if filter == "" {
				continue
			}

			pf.AddIdentityFilter(filter)
		}

		return
	}
}

// CompoundFilter processes a series of strings as compound filters
func CompoundFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			if filter == "" {
				continue
			}

			pf.AddCompoundFilter(filter)
		}

		return
	}
}

// FactFilter processes a series of strings as fact filters
func FactFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			if filter == "" {
				continue
			}

			ff, err := ParseFactFilterString(filter)
			if err != nil {
				return err
			}

			err = pf.AddFactFilter(ff.Fact, ff.Operator, ff.Value)
			if err != nil {
				return err
			}
		}

		return
	}
}

// CombinedFilter processes a series of strings as combined fact and class filters
func CombinedFilter(f ...string) Filter {
	return func(pf *protocol.Filter) (err error) {
		for _, filter := range f {
			parts := strings.Split(filter, " ")

			for _, part := range parts {
				if part == "" {
					continue
				}

				ff, err := ParseFactFilterString(part)
				if err != nil {
					pf.AddClassFilter(part)
					continue
				}

				pf.AddFactFilter(ff.Fact, ff.Operator, ff.Value)
			}

		}

		return
	}
}

// ParseFactFilterString parses a fact filter string as typically typed on the CLI
func ParseFactFilterString(f string) (pf *protocol.FactFilter, err error) {
	pf = &protocol.FactFilter{}

	ff, err := facts.ParseFactFilterString(f)
	if err != nil {
		return nil, err
	}

	pf.Fact = ff[0]
	pf.Operator = ff[1]
	pf.Value = ff[2]

	if pf.Fact == "" || pf.Operator == "" || pf.Value == "" {
		return nil, fmt.Errorf("could not parse fact %s it does not appear to be in a valid format", f)
	}

	return
}

// NewFilter creates a new filter based on the supplied string representations
func NewFilter(fs ...Filter) (f *protocol.Filter, err error) {
	f = protocol.NewFilter()

	for _, filter := range fs {
		err = filter(f)
		if err != nil {
			return
		}
	}

	return
}
