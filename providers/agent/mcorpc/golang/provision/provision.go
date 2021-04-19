package provision

import (
	"runtime"
	"sync"
	"time"

	"github.com/choria-io/go-choria/plugin"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/sirupsen/logrus"
)

// Reply is a generic reply used by most actions
type Reply struct {
	Message string `json:"message"`
}

var mu = &sync.Mutex{}
var allowRestart = true
var log *logrus.Entry

var metadata = &agents.Metadata{
	Name:        "choria_provision",
	Description: "Choria Provisioner",
	Author:      "R.I.Pienaar <rip@devco.net>",
	Version:     "0.3.1",
	License:     "Apache-2.0",
	Timeout:     20,
	URL:         "http://choria.io",
}

var restartCb = restart

func init() {
	switch {
	case runtime.GOOS == "windows":
		SetRestartAction(restartViaExit)
	default:
		SetRestartAction(restart)
	}

}

// New creates a new instance of the agent
func New(mgr server.AgentManager) (agents.Agent, error) {
	log = mgr.Logger()

	agent := mcorpc.New("choria_provision", metadata, mgr.Choria(), log)

	agent.SetActivationChecker(func() bool {
		return mgr.Choria().ProvisionMode() && mgr.Choria().BuildInfo().ProvisionAgent()
	})

	agent.MustRegisterAction("gencsr", csrAction)
	agent.MustRegisterAction("configure", configureAction)
	agent.MustRegisterAction("restart", restartAction)
	agent.MustRegisterAction("reprovision", reprovisionAction)
	agent.MustRegisterAction("release_update", releaseUpdateAction)
	agent.MustRegisterAction("jwt", jwtAction)

	return agent, nil
}

// ChoriaPlugin creates the choria plugin hooks
func ChoriaPlugin() plugin.Pluggable {
	return mcorpc.NewChoriaAgentPlugin(metadata, New)
}

// SetRestartAction sets a custom restart function to call than the default that
// causes a os.Exec() to be issued replacing the running instance with a new
// process on the old pid
func SetRestartAction(f func(splay time.Duration, si agents.ServerInfoSource, log *logrus.Entry)) {
	mu.Lock()
	restartCb = f
	mu.Unlock()
}
