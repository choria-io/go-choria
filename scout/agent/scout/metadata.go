package agent

import (
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/server/agents"
)

var metadata = &agents.Metadata{
	Name:        "scout",
	Description: "Choria Scout API",
	Author:      "R.I.Pienaar <rip@devco.net>",
	Version:     build.Version,
	License:     "Apache-2.0",
	Timeout:     20,
	URL:         "http://choria.io",
}
