// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/rpcutil"
	"github.com/choria-io/go-choria/providers/discovery/inventory"
)

type tInventoryCommand struct {
	command

	fo *discovery.StandardOptions

	file     string
	validate bool
	update   bool
}

func (e *tInventoryCommand) Setup() (err error) {
	if tool, ok := cmdWithFullCommand("tool"); ok {
		e.cmd = tool.Cmd().Command("inventory", "Manage inventory files")
		e.cmd.Arg("file", "File to act one").StringVar(&e.file)
		e.cmd.Flag("validate", "Just validate that the file is valid").UnNegatableBoolVar(&e.validate)
		e.cmd.Flag("update", "Updates an existing inventory file with discovered nodes").UnNegatableBoolVar(&e.update)

		e.fo = discovery.NewStandardOptions()
		e.fo.AddFilterFlags(e.cmd)
		e.fo.AddFlatFileFlags(e.cmd)
		e.fo.AddSelectionFlags(e.cmd)
	}

	return nil
}

func (e *tInventoryCommand) Configure() error {
	return commonConfigure()
}

func (e *tInventoryCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	e.fo.SetDefaultsFromChoria(c)

	switch {
	case e.validate:
		return e.validateFile()
	default:
		return e.updateFile()
	}
}

func (e *tInventoryCommand) updateFile() error {
	var dat = &inventory.DataFile{
		Schema: inventory.DataSchema,
		Groups: []inventory.Group{
			{Name: "all", Filter: &inventory.GroupFilter{Identities: []string{"/./"}}},
		},
	}

	e.file, err = filepath.Abs(e.file)
	if err != nil {
		return err
	}

	if util.FileExist(e.file) && !e.update {
		return fmt.Errorf("%s already exist, pass --update to overwrite its nodes", e.file)
	}

	if util.FileExist(e.file) {
		dat, err = inventory.ReadInventory(e.file, false)
		if err != nil {
			return err
		}
		dat.Nodes = []inventory.Node{}
	}

	err = e.updateInventory(dat)
	if err != nil {
		fmt.Printf("Error updating node data, not saving updated inventory: %s", err)
		os.Exit(1)
	}

	err = e.saveData(dat)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %d nodes to %s\n", len(dat.Nodes), e.file)

	return nil
}

func (e *tInventoryCommand) updateInventory(dat *inventory.DataFile) error {
	rpcc, err := rpcutilclient.New(c, rpcutilclient.Progress(), rpcutilclient.Logger(c.Logger("inventory")), rpcutilclient.Discovery(rpcutilclient.NewMetaNS(e.fo, false)))
	if err != nil {
		return err
	}

	res, err := rpcc.Inventory().Do(ctx)
	if err != nil {
		return err
	}

	nr := res.Stats().NoResponseFrom()
	lnr := len(nr)
	if lnr > 0 {
		fmt.Printf("No responses from %d nodes, not updating inventory:", lnr)
		if lnr > 10 {
			nr = nr[0:10]
		}
		for _, n := range nr {
			fmt.Printf("   %s", n)
		}
		if lnr > 10 {
			fmt.Printf("and %d more not shown", lnr-10)
		}
		os.Exit(1)
	}

	errs := 0
	res.EachOutput(func(r *rpcutilclient.InventoryOutput) {
		if !r.ResultDetails().OK() {
			log.Errorf("Invalid reply from %s: %s", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage())
			errs++
			return
		}

		node := inventory.Node{}
		inventory := rpcutil.InventoryReply{}
		err = r.ParseInventoryOutput(&inventory)
		if err != nil {
			log.Errorf("Invalid reply from %s: %s", r.ResultDetails().Sender(), err)
			errs++
			return
		}

		node.Name = r.ResultDetails().Sender()
		err = json.Unmarshal(inventory.Facts, &node.Facts)
		if err != nil {
			log.Errorf("Invalid reply from %s: %s", r.ResultDetails().Sender(), err)
			errs++
			return
		}

		node.Classes = inventory.Classes
		node.Collectives = inventory.Collectives
		node.Agents = inventory.Agents
		dat.Nodes = append(dat.Nodes, node)
	})
	if errs > 0 {
		return fmt.Errorf("%d errors", errs)
	}

	return nil
}

func (e *tInventoryCommand) saveData(dat *inventory.DataFile) error {
	var jdat []byte
	switch filepath.Ext(e.file) {
	case ".json":
		jdat, err = json.MarshalIndent(dat, "", "  ")
	case ".yaml", ".yml":
		jdat, err = yaml.Marshal(dat)
	default:
		return fmt.Errorf("cannot determine file type from extension")
	}
	if err != nil {
		return err
	}

	tf, err := os.CreateTemp(filepath.Dir(e.file), "")
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(tf, string(jdat))
	if err != nil {
		return err
	}

	tf.Close()
	return os.Rename(tf.Name(), e.file)
}

func (e *tInventoryCommand) validateFile() error {
	if !util.FileExist(e.file) {
		return fmt.Errorf("%s does not exist", e.file)
	}

	dat, err := os.ReadFile(e.file)
	if err != nil {
		return err
	}

	ext := filepath.Ext(e.file)
	if ext == ".yaml" || ext == ".yml" {
		dat, err = yaml.YAMLToJSON(dat)
		if err != nil {
			return err
		}
	}

	warnings, err := inventory.ValidateInventory(dat)
	if err != nil {
		return err
	}

	if len(warnings) == 0 {
		fmt.Printf("%s is a valid inventory file\n", e.file)
		return nil
	}

	fmt.Printf("%s is not a valid inventory file:\n\n", e.file)
	for _, w := range warnings {
		fmt.Printf("\t%s\n", w)
	}
	os.Exit(1)

	return nil

}

func init() {
	cli.commands = append(cli.commands, &tInventoryCommand{})
}
