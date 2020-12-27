// +build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/choria-io/go-choria/generators/client"
)

var ddls map[string]string

var ddlt = `
// generated code; DO NOT EDIT

package ddlcache

import (
        "encoding/base64"
        "encoding/json"
		"fmt"

        "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

var ddls = map[string]string{
{{- range $key, $val := . }}
	"{{ $key }}": "{{ $val }}",
{{- end }}
}

// DDLBytes is the raw JSON encoded DDL file for the agent
func DDLBytes(agent string) ([]byte, error) {
		ddl,ok:=ddls[agent]
		if !ok {
			return nil, fmt.Errorf("unknown agent %s", agent)
		}

        return base64.StdEncoding.DecodeString(ddl)
}

// DDL is a parsed and loaded DDL for the agent a
func DDL(a string) (*agent.DDL, error) {
        ddlj, err := DDLBytes(a)
        if err != nil {
                return nil, err
        }

        ddl := &agent.DDL{}
        err = json.Unmarshal(ddlj, ddl)
        if err != nil {
                return nil, err
        }

        return ddl, nil
}
`

func generate(agent string, ddl string, pkg string) error {
	if ddl == "" {
		ddl = fmt.Sprintf("client/ddlcache/%s.json", agent)
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

	rawddl, err := ioutil.ReadFile(ddl)
	if err != nil {
		return err
	}

	ddls[agent] = base64.StdEncoding.EncodeToString(rawddl)

	return nil
}

func main() {
	ddls = make(map[string]string)

	for _, agent := range []string{"rpcutil", "choria_util", "scout"} {
		err := generate(agent, "", "")
		if err != nil {
			panic(err)
		}
	}

	out, err := os.OpenFile("client/ddlcache/cache.go", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	t := template.New("ddlcache")
	templ, err := t.Parse(ddlt)
	if err != nil {
		panic(err)
	}

	err = templ.Execute(out, ddls)
	if err != nil {
		panic(err)
	}

	out.Close()
	err = client.FormatGoSource(out.Name())
	if err != nil {
		panic(err)
	}
}
