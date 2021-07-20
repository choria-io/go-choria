// +build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/choria-io/go-choria/generators/client"
)

var ddls map[string]string

func generate(agent string, ddl string, pkg string) error {
	if ddl == "" {
		ddl = fmt.Sprintf("internal/fs/ddl/cache/agent/%s.json", agent)
	}

	if pkg == "" {
		pkg = agent + "client"
	}

	g := &client.Generator{
		DDLFile:     ddl,
		OutDir:      fmt.Sprintf("client/%sclient", agent),
		PackageName: pkg,
	}

	err := os.RemoveAll(g.OutDir)
	if err != nil {
		return err
	}

	err = os.Mkdir(g.OutDir, 0775)
	if err != nil {
		return err
	}

	err = g.GenerateClient()
	if err != nil {
		return err
	}

	rawddl, err := os.ReadFile(ddl)
	if err != nil {
		return err
	}

	ddls[agent] = base64.StdEncoding.EncodeToString(rawddl)

	return nil
}

func main() {
	ddls = make(map[string]string)

	for _, agent := range []string{"rpcutil", "choria_util", "scout", "choria_provision", "choria_registry"} {
		err := generate(agent, "", "")
		if err != nil {
			panic(err)
		}
	}
}
