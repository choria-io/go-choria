package provision

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

type ReprovisionRequest struct {
	Token string `json:"token"`
}

func reprovisionAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if agent.Choria.ProvisionMode() {
		abort("Server is already in provisioning mode, cannot enable provisioning mode again", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine the configuration file to manage", reply)
		return
	}

	args := ReprovisionRequest{}
	if !mcorpc.ParseRequestData(&args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	cfg := make(map[string]string)

	cfg["plugin.choria.server.provision"] = "1"
	cfg["loglevel"] = "debug"

	if agent.Config.LogFile != "" {
		cfg["logfile"] = agent.Config.LogFile
	}

	if build.ProvisionRegistrationData == "" && agent.Config.Choria.FileContentRegistrationData != "" {
		cfg["registration"] = "file_content"
		cfg["plugin.choria.registration.file_content.data"] = agent.Config.Choria.FileContentRegistrationData
	}

	_, err := writeConfig(cfg, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	splay := time.Duration(rand.Intn(10)+2) * time.Second
	go restartCb(splay, agent.ServerInfoSource, agent.Log)

	reply.Data = Reply{fmt.Sprintf("Restarting after %v", splay)}
}
