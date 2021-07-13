package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

type kvPutCommand struct {
	command
	name string
	key  string
	val  string
}

func (k *kvPutCommand) Setup() error {
	if kv, ok := cmdWithFullCommand("kv"); ok {
		k.cmd = kv.Cmd().Command("put", "Puts a value")
		k.cmd.Arg("bucket", "The bucket name").Required().StringVar(&k.name)
		k.cmd.Arg("key", "The key to delete").Required().StringVar(&k.key)
		k.cmd.Arg("val", "The value to store, - for STDIN").Required().StringVar(&k.val)
	}

	return nil
}

func (k *kvPutCommand) Configure() error {
	return commonConfigure()
}

func (k *kvPutCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	_, store, err := c.KV(ctx, k.name)
	if err != nil {
		return err
	}

	val := []byte(k.val)
	if k.val == "-" {
		val, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	}

	_, err = store.Put(k.key, val)
	if err != nil {
		return err
	}

	fmt.Println(string(val))

	return nil
}

func init() {
	cli.commands = append(cli.commands, &kvPutCommand{})
}
