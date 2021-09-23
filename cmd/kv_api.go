package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/config"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/kv"
)

type kvAPICommand struct {
	command
	update bool
	list   bool
	delete bool
	check  bool

	name          string
	history       uint64
	ttl           int
	replicas      int
	maxValueSize  int32
	maxBucketSize int64
	force         bool
}

func (g *kvAPICommand) Setup() (err error) {
	if gov, ok := cmdWithFullCommand("kv"); ok {
		g.cmd = gov.Cmd().Command("api", "API to manage Key-Value buckets via JSON inputs and outputs").Hidden()
		g.cmd.Flag("ensure", "Creates or Updates the bucket based on supplied configuration").BoolVar(&g.update)
		g.cmd.Flag("delete", "Deletes a specific bucket").PlaceHolder("GOVERNOR").BoolVar(&g.delete)
		g.cmd.Flag("list", "List known buckets").BoolVar(&g.list)
		g.cmd.Flag("check", "Checks if the API is available").BoolVar(&g.check)

		g.cmd.Flag("name", "KV Bucket name").PlaceHolder("NAME").StringVar(&g.name)
		g.cmd.Flag("history", "How many historic values to keep for each key").PlaceHolder("CAPACITY").Uint64Var(&g.history)
		g.cmd.Flag("expire", "Expire values from the bucket after this duration").PlaceHolder("SECONDS").IntVar(&g.ttl)
		g.cmd.Flag("replicas", "How many replicas to store on the server").PlaceHolder("REPLICAS").IntVar(&g.replicas)
		g.cmd.Flag("max-value-size", "Maximum size of any value in the bucket").PlaceHolder("BYTES").Int32Var(&g.maxValueSize)
		g.cmd.Flag("max-bucket-size", "Maximum size for the entire bucket").PlaceHolder("BYTES").Int64Var(&g.maxBucketSize)
		g.cmd.Flag("force", "Force changes that require the bucket to be recreated").BoolVar(&g.force)
	}

	return nil
}

func (g *kvAPICommand) Configure() error {
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

func (g *kvAPICommand) Run(wg *sync.WaitGroup) (err error) {
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

func (g *kvAPICommand) updateCmd() {
	switch {
	case g.name == "":
		g.fail("name required")
	case g.history == 0:
		g.fail("history can not be 0")
	case g.replicas < 1 || g.replicas > 5:
		g.fail("replicas should be 1-5")
	}

	ttl := time.Duration(g.ttl) * time.Second
	opts := []kv.Option{
		kv.WithHistory(g.history),
		kv.WithTTL(ttl),
		kv.WithReplicas(uint(g.replicas)),
		kv.WithMaxBucketSize(g.maxBucketSize),
		kv.WithMaxValueSize(g.maxValueSize)}

	bucket, conn, err := c.KVWithConn(ctx, nil, g.name, true, opts...)
	if err != nil {
		g.fail("update failed: %s", err)
	}

	status, err := bucket.Status()
	if err != nil {
		g.fail("update failed: %s", err)
	}

	ok, failed := status.Replicas()
	needsRecreate := ok+failed != g.replicas
	needUpdate := status.TTL() != ttl || status.History() != int64(g.history) || status.MaxValueSize() != g.maxValueSize || status.MaxBucketSize() != g.maxBucketSize

	if needsRecreate {
		if !g.force {
			g.fail("changing replicas requires force option")
		}
		err = bucket.Destroy()
		if err != nil {
			g.fail("could not remove bucket to update replicas: %s", err)
		}

		bucket, err = c.KV(ctx, conn, g.name, true, opts...)
		if err != nil {
			g.fail("update failed: %s", err)
		}

		needUpdate = false
	}

	if needUpdate {
		mgr, err := jsm.New(conn.Nats())
		if err != nil {
			g.fail("update failed: %s", err)
		}

		str, err := mgr.LoadStream(status.BackingStore())
		if err != nil {
			g.fail("update failed: %s", err)
		}

		cfg := str.Configuration()
		cfg.MaxAge = ttl
		cfg.MaxMsgsPer = int64(g.history)
		cfg.MaxMsgSize = g.maxValueSize
		cfg.MaxBytes = g.maxBucketSize

		err = str.UpdateConfiguration(cfg)
		if err != nil {
			g.fail("update failed: %s", err)
		}

		bucket, err = c.KV(ctx, conn, g.name, true, opts...)
		if err != nil {
			g.fail("update failed: %s", err)
		}
	}

	status, err = bucket.Status()
	if err != nil {
		g.fail("update failed: %s", err)
	}
	ok, failed = status.Replicas()

	g.jsonDump(map[string]interface{}{
		"name":            status.Bucket(),
		"history":         status.History(),
		"expire":          status.TTL().Seconds(),
		"replicas":        ok + failed,
		"max_value_size":  status.MaxValueSize(),
		"max_bucket_size": status.MaxBucketSize(),
	})
}

func (g *kvAPICommand) deleteCmd() {
	if g.name == "" {
		g.fail("no name given")
	}

	bucket, err := c.KV(ctx, nil, g.name, false)
	if err != nil {
		g.fail("loading bucket failed: %s", err)
	}

	err = bucket.Destroy()
	if err != nil {
		g.fail("delete failed: %s", err)
	}
}

func (g *kvAPICommand) listCmd() {
	type bucket struct {
		Name          string `json:"name"`
		History       int    `json:"history"`
		TTL           int    `json:"expire"`
		Replicas      int    `json:"replicas"`
		MaxValueSize  int    `json:"max_value_size"`
		MaxBucketSize int    `json:"max_bucket_size"`
	}

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, "kv api: list", c.Logger("kv"))
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	known, err := mgr.StreamNames(&jsm.StreamNamesFilter{
		Subject: "$KV.>",
	})
	if err != nil {
		g.fail("connection failed: %s", err)
	}

	var buckets = []bucket{}
	for i := 0; i < len(known); i++ {
		if !strings.HasPrefix(known[i], "KV_") {
			continue
		}

		kv, err := c.KV(ctx, nil, strings.TrimPrefix(known[i], "KV_"), false)
		if err != nil {
			g.fail("loading buckets failed: %s", err)
		}

		status, err := kv.Status()
		if err != nil {
			g.fail("loading buckets failed: %s", err)
		}

		ok, failed := status.Replicas()
		buckets = append(buckets, bucket{
			Name:          status.Bucket(),
			History:       int(status.History()),
			TTL:           int(status.TTL().Seconds()),
			Replicas:      ok + failed,
			MaxValueSize:  int(status.MaxValueSize()),
			MaxBucketSize: int(status.MaxBucketSize()),
		})
	}

	g.jsonDump(buckets)
}

func (g *kvAPICommand) fail(format string, a ...interface{}) {
	g.jsonDump(map[string]string{
		"error": fmt.Sprintf(format, a...),
	})

	os.Exit(1)
}

func (g *kvAPICommand) jsonDump(d interface{}) {
	j, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(j))
}
func init() {
	cli.commands = append(cli.commands, &kvAPICommand{})
}
