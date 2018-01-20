package serverinfotest

import (
	"encoding/json"
	"time"

	"github.com/choria-io/go-choria/server/agents"
)

type InfoSource struct{}

func (si *InfoSource) KnownAgents() []string {
	return []string{"stub_agent"}
}

func (si *InfoSource) AgentMetadata(a string) (agents.Metadata, bool) {
	return agents.Metadata{
		Author:      "stub@example.net",
		Description: "Stub Agent",
		License:     "Apache-2.0",
		Name:        "stub_agent",
		Timeout:     10,
		URL:         "https://choria.io/",
		Version:     "1.0.0",
	}, true
}

func (si *InfoSource) ConfigFile() string {
	return "/stub/config.cfg"
}

func (si *InfoSource) Classes() []string {
	return []string{"one", "two"}
}

func (si *InfoSource) Facts() json.RawMessage {
	return json.RawMessage(`{"stub":true}`)
}

func (si *InfoSource) StartTime() time.Time {
	return time.Now()
}

func (si *InfoSource) Stats() agents.ServerStats {
	return agents.ServerStats{}
}
