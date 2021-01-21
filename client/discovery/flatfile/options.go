package flatfile

import (
	"io"

	"github.com/choria-io/go-choria/protocol"
)

type SourceFormat int

const (
	// TextFormat reads nodes from a text file 1 node per line
	TextFormat = iota + 1

	// JSONFormat parses a JSON file expecting an array of nodes
	JSONFormat

	// YAMLFormat parses a YAML file expecting an array of nodes
	YAMLFormat

	// ChoriaResponsesFormat uses Choria responses as produced by choria req -j as source
	ChoriaResponsesFormat
)

type dOpts struct {
	source string
	format SourceFormat
	reader io.Reader
	filter *protocol.Filter
	do     map[string]string
}

// DiscoverOption configures the broadcast discovery method
type DiscoverOption func(o *dOpts)

// Filter sets the filter to use for the discovery, else a blank one is used
func Filter(f *protocol.Filter) DiscoverOption {
	return func(o *dOpts) {
		o.filter = f
	}
}

// Format specifies the file format
func Format(f SourceFormat) DiscoverOption {
	return func(o *dOpts) {
		o.format = f
	}
}

// File sets the file to read nodes from
func File(f string) DiscoverOption {
	return func(o *dOpts) {
		o.source = f
	}
}

// Reader specifies a io.Reader as source
func Reader(r io.Reader) DiscoverOption {
	return func(o *dOpts) {
		o.reader = r
	}
}

// DiscoveryOptions sets the key value pairs that make user supplied discovery options.
//
// Supported options:
//
//   filter - GJSON Path Syntax search over YAML or JSON data
func DiscoveryOptions(opt map[string]string) DiscoverOption {
	return func(o *dOpts) {
		o.do = opt
	}
}
