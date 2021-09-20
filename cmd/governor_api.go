package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/governor"
)

type tGovAPICommand struct {
	command
	update bool
	list   bool
	delete bool
	check  bool

	collective string
	name       string
	limit      int64
	expire     int
	replicas   int
	force      bool
}

func (g *tGovAPICommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("governor"); ok {
		g.cmd = gov.Cmd().Command("api", "API to manage Governors via JSON inputs and outputs").Hidden()
		g.cmd.Flag("ensure", "Creates or Updates the governor based on supplied configuration").BoolVar(&g.update)
		g.cmd.Flag("delete", "Deletes a specific governor").PlaceHolder("GOVERNOR").BoolVar(&g.delete)
		g.cmd.Flag("list", "List known governors").BoolVar(&g.list)
		g.cmd.Flag("check", "Checks if the API is available").BoolVar(&g.check)

		g.cmd.Flag("name", "Governor name").PlaceHolder("NAME").StringVar(&g.name)
		g.cmd.Flag("capacity", "Governor capacity").PlaceHolder("CAPACITY").Int64Var(&g.limit)
		g.cmd.Flag("expire", "How long before entries expire from the governor").PlaceHolder("SECONDS").IntVar(&g.expire)
		g.cmd.Flag("replicas", "How many replicas to store on the server").PlaceHolder("REPLICAS").IntVar(&g.replicas)
		g.cmd.Flag("collective", "The sub-collective to install the Governor in").PlaceHolder("COLLECTIVE").StringVar(&g.collective)
		g.cmd.Flag("force", "Force changes that require the governor to be recreated").BoolVar(&g.force)
	}

	return nil
}

func (g *tGovAPICommand) Configure() error {
	if os.Getuid() == 0 {
		cfg, err = config.NewSystemConfig(configFile, true)
		if err != nil {
			g.fail("config failed: %s", err)
		}
		cfg.LogLevel = "error"
	} else {
		err = commonConfigure()
		if err != nil {
			g.fail("config failed: %s", err)
		}
	}

	return nil
}

func (g *tGovAPICommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	switch {
	case g.check:
		g.jsonDump(map[string]string{"api": "ok"})
	case g.update:
		g.updateCmd()
	case g.delete:
		g.deleteCmd()
	case g.list:
		g.listCmd()
	default:
		g.fail("no command given")
	}

	return nil
}

func (g *tGovAPICommand) updateCmd() {
	switch {
	case g.name == "":
		g.fail("name required")
	case g.limit == 0:
		g.fail("capacity can not be 0")
	case g.expire == 0:
		g.fail("expire can not be 0")
	case g.replicas < 1 || g.replicas > 5:
		g.fail("replicas should be 1-5")
	case g.collective == "":
		g.fail("collective is required")
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", "governor_list"), c.Logger("governor"))
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	gov, err := governor.NewJSGovernorManager(g.name, uint64(g.limit), time.Duration(g.expire)*time.Second, uint(g.replicas), mgr, true, governor.WithSubject(util.GovernorSubject(g.name, g.collective)))
	if err != nil {
		g.fail("update failed: %s", err)
	}

	if gov.Replicas() != g.replicas {
		if !g.force {
			g.fail("replica update requires force")
		}

		err = gov.Stream().Delete()
		if err != nil {
			g.fail("deleting existing stream failed: %s", err)
		}

		gov, err = governor.NewJSGovernorManager(g.name, uint64(g.limit), time.Duration(g.expire)*time.Second, uint(g.replicas), mgr, true, governor.WithSubject(util.GovernorSubject(g.name, g.collective)))
		if err != nil {
			g.fail("update failed: %s", err)
		}
	}

	parts := strings.Split(gov.Subject(), ".")

	g.jsonDump(map[string]interface{}{
		"name":       gov.Name(),
		"capacity":   gov.Limit(),
		"expire":     gov.MaxAge().Seconds(),
		"replicas":   gov.Replicas(),
		"collective": parts[0],
	})
}

func (g *tGovAPICommand) deleteCmd() {
	if g.name == "" {
		g.fail("no name given")
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", "governor_list"), c.Logger("governor"))
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	str, err := mgr.LoadStream(fmt.Sprintf("GOVERNOR_%s", g.name))
	if err != nil {
		if jsm.IsNatsError(err, 10059) {
			return
		}
		g.fail("could not find governor: %s", err)
	}

	err = str.Delete()
	if err != nil {
		g.fail("delete failed: %s", err)
	}
}

func (g *tGovAPICommand) listCmd() {
	type gov struct {
		Name       string `json:"name"`
		Capacity   int64  `json:"capacity"`
		Expire     int    `json:"expire"`
		Replicas   int    `json:"replicas"`
		Collective string `json:"collective"`
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("governor manager: %s", "governor_list"), c.Logger("governor"))
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	known, err := mgr.StreamNames(&jsm.StreamNamesFilter{
		Subject: util.GovernorSubject("*", "*"),
	})
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	var govs = make([]gov, len(known))
	for i := 0; i < len(known); i++ {
		name := strings.TrimPrefix(known[i], "GOVERNOR_")

		mgr, err := governor.NewJSGovernorManager(name, 0, 0, 1, mgr, false)
		if err != nil {
			g.fail("loading failed: %s", err)
		}

		parts := strings.Split(mgr.Subject(), ".")
		govs[i] = gov{
			Name:       name,
			Capacity:   mgr.Limit(),
			Expire:     int(mgr.MaxAge().Seconds()),
			Replicas:   mgr.Replicas(),
			Collective: parts[0],
		}
	}

	g.jsonDump(govs)
}

func (g *tGovAPICommand) fail(format string, a ...interface{}) {
	g.jsonDump(map[string]string{
		"error": fmt.Sprintf(format, a...),
	})

	os.Exit(1)
}

func (g *tGovAPICommand) jsonDump(d interface{}) {
	j, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(j))
}
func init() {
	cli.commands = append(cli.commands, &tGovAPICommand{})
}
