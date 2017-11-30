package discovery

import (
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/version"
)

type Agent struct {
	meta *agents.Metadata
	log  *logrus.Entry
}

func New(log *logrus.Entry) (*Agent, error) {
	a := &Agent{
		log: log.WithFields(logrus.Fields{"agent": "discovery"}),
		meta: &agents.Metadata{
			Name:        "discovery",
			Description: "Discovery Agent",
			Author:      "R.I.Pienaar <rip@devco.net>",
			Version:     version.Version,
			License:     version.License,
			Timeout:     2,
			URL:         "http://choria.io",
		},
	}

	return a, nil
}

func (da *Agent) Name() string {
	return da.meta.Name
}

func (da *Agent) Metadata() *agents.Metadata {
	return da.meta
}

func (da *Agent) HandleAgentMsg(msg *choria.Message) (*[]byte, error) {
	if strings.Contains(msg.Payload, "ping") {
		r := []byte("pong")
		return &r, nil
	}

	return nil, fmt.Errorf("unknown request: %s", msg)
}
