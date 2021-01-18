package flatfile

import (
	"io"
)

type SourceFormat int

const (
	// TextFormat reads nodes from a text file 1 node per line
	TextFormat = iota + 1

	// JSONFormat parses a JSON file expecting an array of nodes
	JSONFormat

	// YAMLFormat parses a YAML file expecting an array of nodes
	YAMLFormat

	// ChoriaResponses uses Choria responses as produced by choria req -j as source
	ChoriaResponses
)

type dOpts struct {
	source string
	format SourceFormat
	reader io.Reader
}

// DiscoverOption configures the broadcast discovery method
type DiscoverOption func(o *dOpts)

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
