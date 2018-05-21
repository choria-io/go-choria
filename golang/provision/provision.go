package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

type ConfigureRequest struct {
	Configuration map[string]string `json:"config"`
}

type RestartRequest struct {
	Splay int `json:"splay"`
}

type Reply struct {
	Message string `json:"message"`
}

var mu = &sync.Mutex{}
var allowRestart = true

func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	metadata := &agents.Metadata{
		Name:        "choria_provision",
		Description: "Choria Provisioner",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     build.Version,
		License:     build.License,
		Timeout:     2,
		URL:         "http://choria.io",
	}

	agent := mcorpc.New("choria_provision", metadata, mgr.Choria(), mgr.Logger())

	agent.MustRegisterAction("configure", configureAction)
	agent.MustRegisterAction("restart", restartAction)
	agent.MustRegisterAction("reprovision", reprovisionAction)

	return agent, nil
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

	cfg := make(map[string]string)

	cfg["plugin.choria.server.provision"] = "1"
	cfg["loglevel"] = "debug"

	if agent.Config.LogFile != "" {
		cfg["logfile"] = agent.Config.LogFile
	}

	if agent.Config.Choria.FileContentRegistrationData != "" {
		cfg["registration"] = "file_content"
		cfg["plugin.choria.registration.file_content.data"] = agent.Config.Choria.FileContentRegistrationData
	}

	_, err := writeConfig(cfg, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	splay := time.Duration(rand.Intn(10) + 2)

	if allowRestart {
		go restart(splay, agent.Log)
	}

	reply.Data = Reply{fmt.Sprintf("Restarting after %ds", splay)}
}

func configureAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot reconfigure a server that is not in provisioning mode", reply)
		return
	}

	if agent.Config.ConfigFile == "" {
		abort("Cannot determine the configuration file to manage", reply)
		return
	}

	args := ConfigureRequest{}
	err := json.Unmarshal(req.Data, &args)
	if err != nil {
		abort(fmt.Sprintf("Could not parse request arguments: %s", err), reply)
		return
	}

	if len(args.Configuration) == 0 {
		abort("Did not receive any configuration to write, cannot write a empty configuration file", reply)
		return
	}

	lines, err := writeConfig(args.Configuration, req, agent.Config, agent.Log)
	if err != nil {
		abort(fmt.Sprintf("Could not write config: %s", err), reply)
		return
	}

	reply.Data = Reply{fmt.Sprintf("Wrote %d lines to %s", lines, agent.Config.ConfigFile)}
}

func restartAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() {
		abort("Cannot restart a server that is not in provisioning mode", reply)
		return
	}

	args := RestartRequest{}
	err := json.Unmarshal(req.Data, &args)
	if err != nil {
		abort(fmt.Sprintf("Could not parse request arguments: %s", err), reply)
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

	splay := time.Duration(rand.Intn(args.Splay) + 2)

	agent.Log.Warnf("Restarting server via request %s from %s (%s) with splay %d", req.RequestID, req.CallerID, req.SenderID, splay)

	if allowRestart {
		go restart(splay, agent.Log)
	}

	reply.Data = Reply{fmt.Sprintf("Restarting Choria Server after %ds", splay)}
}

func abort(msg string, reply *mcorpc.Reply) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = msg
}

func writeConfig(settings map[string]string, req *mcorpc.Request, cfg *config.Config, log *logrus.Entry) (int, error) {
	cfile := cfg.ConfigFile

	_, err := os.Stat(cfile)
	if err == nil {
		cfile, err = filepath.EvalSymlinks(cfg.ConfigFile)
		if err != nil {
			return 0, fmt.Errorf("cannot determine full path to config file %s: %s", cfile, err)
		}
	}

	log.Warnf("Rewriting configuration file %s in request %s from %s (%s)", cfile, req.RequestID, req.CallerID, req.SenderID)

	cdir := filepath.Dir(cfile)

	tmpfile, err := ioutil.TempFile(cdir, "provision")
	if err != nil {
		return 0, fmt.Errorf("cannot create a temp file in %s: %s", cdir, err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = fmt.Fprintf(tmpfile, "# configuration file writen in request %s from %s (%s) at %s\n", req.RequestID, req.CallerID, req.SenderID, time.Now())
	if err != nil {
		return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
	}

	written := 1

	for k, v := range settings {
		log.Infof("Adding configuration: %s = %s", k, v)

		_, err := fmt.Fprintf(tmpfile, "%s=%s\n", k, v)
		if err != nil {
			return 0, fmt.Errorf("could not write to temp file %s: %s", tmpfile.Name(), err)
		}

		written++
	}

	err = tmpfile.Close()
	if err != nil {
		return 0, fmt.Errorf("could not close temp file %s: %s", tmpfile.Name(), err)
	}

	_, err = config.NewConfig(tmpfile.Name())
	if err != nil {
		return 0, fmt.Errorf("generated configuration could not be parsed: %s", err)
	}

	err = os.Rename(tmpfile.Name(), cfile)
	if err != nil {
		return 0, fmt.Errorf("could not rename temp file %s to %s: %s", tmpfile.Name(), cfile, err)
	}

	return written, nil
}

func restart(splay time.Duration, log *logrus.Entry) {
	mu.Lock()
	defer mu.Unlock()

	log.Warnf("Restarting Choria Server after %ds splay time", splay)
	time.Sleep(splay * time.Second)

	err := syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		log.Errorf("Could not restart server: %s", err)
	}
}
