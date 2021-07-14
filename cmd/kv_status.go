package cmd

import (
	"fmt"
	"sync"
)

type kvStatusCommand struct {
	command
	name string
}

func (k *kvStatusCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("status", "View the status of a bucket").Alias("info")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
	}

	return nil
}

func (k *kvStatusCommand) Configure() error {
	return commonConfigure()
}

func (k *kvStatusCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	store, err := c.KV(ctx, nil, k.name, false)
	if err != nil {
		return err
	}

	status, err := store.Status()
	if err != nil {
		return err
	}

	fmt.Printf("%s Key-Value Store\n", status.Bucket())
	fmt.Println()
	fmt.Printf("     Bucket Name: %s\n", status.Bucket())
	fmt.Printf("   Values Stored: %d\n", status.Values())
	fmt.Printf("         History: %d\n", status.History())
	fmt.Printf("             TTL: %v\n", status.TTL())
	fmt.Printf(" Max Bucket Size: %d\n", status.MaxBucketSize())
	fmt.Printf("  Max Value Size: %d\n", status.MaxValueSize())

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvStatusCommand{})
}
