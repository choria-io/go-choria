package provision

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/lifecycle"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type RestartRequest struct {
	Token string `json:"token"`
	Splay int    `json:"splay"`
}

func restartAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() && build.ProvisionToken == "" {
		abort("Cannot restart a server that is not in provisioning mode or with no token set", reply)
		return
	}

	args := &RestartRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	cfg, err := config.NewConfig(agent.Config.ConfigFile)
	if err != nil {
		abort(fmt.Sprintf("Configuration %s could not be parsed, restart cannot continue: %s", agent.Config.ConfigFile, err), reply)
		return
	}

	if cfg.Choria.Provision {
		abort(fmt.Sprintf("Configuration %s enables provisioning, restart cannot continue", agent.Config.ConfigFile), reply)
		return
	}

	if args.Splay == 0 {
		args.Splay = 10
	}

	splay := time.Duration(rand.Intn(args.Splay)+2) * time.Second
	agent.Log.Warnf("Restarting server via request %s from %s (%s) with splay %v", req.RequestID, req.CallerID, req.SenderID, splay)

	go restartCb(splay, agent.ServerInfoSource, agent.Log)

	reply.Data = Reply{fmt.Sprintf("Restarting Choria Server after %v", splay)}
}

func restartViaExit(splay time.Duration, si agents.ServerInfoSource, log *logrus.Entry) {
	if !allowRestart {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	allowRestart = false

	log.Warnf("Shutting down Choria Server after %v splay time", splay)
	time.Sleep(splay)
	log.Warn("Initiating Choria Server shutdown")

	// sends a shutdown event
	err := si.PrepareForShutdown()
	if err != nil {
		log.Errorf("Could not prepare server for clean shutdown: %s", err)
	}

	os.Exit(0)
}

func restart(splay time.Duration, si agents.ServerInfoSource, log *logrus.Entry) {
	if !allowRestart {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	allowRestart = false

	err := si.NewEvent(lifecycle.Shutdown)
	if err != nil {
		log.Errorf("Could not publish shutdown event: %s", err)
	}

	log.Warnf("Restarting Choria Server after %v splay time", splay)
	time.Sleep(splay)
	log.Warnf("Initiating Choria Server restart using %s", strings.Join(os.Args, " "))

	err = syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		allowRestart = true
		log.Errorf("Could not restart server: %s", err)
	}
}
