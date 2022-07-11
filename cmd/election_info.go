// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats.go"
)

type tElectionInfoCommand struct {
	command

	bucket string
}

func (i *tElectionInfoCommand) Setup() (err error) {
	if elect, ok := cmdWithFullCommand("election"); ok {
		i.cmd = elect.Cmd().Command("info", "View information about an Election bucket")
		i.cmd.Flag("bucket", "Use a specific bucket for elections").Default("CHORIA_LEADER_ELECTION").StringVar(&i.bucket)
	}

	return nil
}

func (i *tElectionInfoCommand) Configure() (err error) {
	return commonConfigure()
}

func (i *tElectionInfoCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	logger := c.Logger("election")

	conn, err := c.NewConnector(ctx, c.MiddlewareServers, fmt.Sprintf("election %s %s", i.bucket, c.Config.Identity), logger)
	if err != nil {
		return err
	}

	js, err := conn.Nats().JetStream()
	if err != nil {
		return err
	}

	kv, err := js.KeyValue(i.bucket)
	if err != nil {
		return fmt.Errorf("cannot access KV Bucket %s: %v", i.bucket, err)
	}

	status, err := kv.Status()
	if err != nil {
		return fmt.Errorf("cannot access KV Bucket %s: %v", i.bucket, err)
	}

	si := status.(*nats.KeyValueBucketStatus).StreamInfo()

	if si.Config.MaxMsgsPerSubject != 1 {
		fmt.Printf("WARNING: %s is not a valid election bucket, historic entries are retained\n\n", status.Bucket())
	}

	switch {
	case si.Config.MaxAge < 5*time.Second:
		fmt.Printf("WARNING: %s is not a valid election bucket, maximum age %v is too low\n\n", status.Bucket(), si.Config.MaxAge)
	case si.Config.MaxAge > time.Hour:
		fmt.Printf("WARNING: %s is not a valid election bucket, maximum age %v is too high\n\n", status.Bucket(), si.Config.MaxAge)
	case si.Config.MaxAge > time.Minute:
		fmt.Printf("WARNING: %s has a very high maximum age of %v, recovery time will be slow\n\n", status.Bucket(), si.Config.MaxAge)
	}

	var peers []string

	if si.Cluster != nil {
		if si.Cluster.Leader == "" {
			fmt.Printf("WARNING: %s does not have a cluster leader\n\n", status.Bucket())
		} else {
			peers = append(peers, si.Cluster.Leader)
		}

		for _, r := range si.Cluster.Replicas {
			switch {
			case r.Offline:
				fmt.Printf("WARNING: Replica peer %s is offline\n\n", r.Name)
			case !r.Current || r.Lag > 100:
				fmt.Printf("WARNING: Replica peer %s is not current with %d replication lag\n\n", r.Name, r.Lag)
			}
			peers = append(peers, r.Name)
		}
		peers = i.compactStrings(peers)
		if si.Cluster.Leader != "" {
			peers[0] = peers[0] + "*"
		}
	}

	fmt.Printf("Election bucket information for %s\n\n", status.Bucket())
	fmt.Printf("       Created: %s\n", si.Created.Format(time.RFC822Z))
	fmt.Printf("       Storage: %s\n", si.Config.Storage.String())
	fmt.Printf("  Maximum Time: %v\n", si.Config.MaxAge)
	if si.Config.Replicas == 1 {
		fmt.Printf("      Replicas: 1\n")
	} else {
		fmt.Printf("      Replicas: %d on hosts %s\n", si.Config.Replicas, strings.Join(peers, ", "))
	}
	fmt.Printf("     Elections: %d\n", si.State.Msgs)

	if si.State.Msgs > 0 {
		table := iu.NewUTF8Table("Election", "Leader")
		table.AddTitle("Active Elections")

		w, err := kv.WatchAll()
		if err != nil {
			return fmt.Errorf("cannot load elections: %v", err)
		}
		for {
			entry := <-w.Updates()
			if entry == nil {
				w.Stop()
				break
			}
			if entry.Operation() == nats.KeyValuePut {
				table.AddRow(entry.Key(), string(entry.Value()))
			}
		}
		fmt.Println()
		fmt.Println(table.Render())
	}

	return nil
}

func (i *tElectionInfoCommand) compactStrings(source []string) []string {
	if len(source) == 0 {
		return source
	}

	hnParts := make([][]string, len(source))
	shortest := math.MaxInt8

	for i, name := range source {
		hnParts[i] = strings.Split(name, ".")
		if len(hnParts[i]) < shortest {
			shortest = len(hnParts[i])
		}
	}

	toRemove := ""

	// we dont chop the 0 item off
	for i := shortest - 1; i > 0; i-- {
		s := hnParts[0][i]

		remove := true
		for _, name := range hnParts {
			if name[i] != s {
				remove = false
				break
			}
		}

		if remove {
			toRemove = "." + s + toRemove
		} else {
			break
		}
	}

	result := make([]string, len(source))
	for i, name := range source {
		result[i] = strings.TrimSuffix(name, toRemove)
	}

	return result
}

func init() {
	cli.commands = append(cli.commands, &tElectionInfoCommand{})
}
