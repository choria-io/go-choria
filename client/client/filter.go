package client

import (
	"github.com/choria-io/go-protocol/filter"
	"github.com/choria-io/go-protocol/protocol"
)

// AgentFilter processes a series of strings as agent filters
func AgentFilter(f ...string) filter.Filter {
	return filter.AgentFilter(f...)
}

// ClassFilter processes a series of strings as agent filters
func ClassFilter(f ...string) filter.Filter {
	return filter.ClassFilter(f...)
}

// IdentityFilter processes a series of strings as identity filters
func IdentityFilter(f ...string) filter.Filter {
	return filter.IdentityFilter(f...)
}

// CompoundFilter processes a series of strings as compound filters
func CompoundFilter(f ...string) filter.Filter {
	return filter.CompoundFilter(f...)
}

// FactFilter processes a series of strings as fact filters
func FactFilter(f ...string) filter.Filter {
	return filter.FactFilter(f...)
}

// CombinedFilter processes a series of strings as combined fact and class filters
func CombinedFilter(f ...string) filter.Filter {
	return filter.CombinedFilter(f...)
}

// ParseFactFilterString parses a fact filter string as typically typed on the CLI
func ParseFactFilterString(f string) (pf *protocol.FactFilter, err error) {
	return filter.ParseFactFilterString(f)
}

// NewFilter creates a new filter based on the supplied string representations
func NewFilter(fs ...filter.Filter) (f *protocol.Filter, err error) {
	return filter.NewFilter(fs...)
}
