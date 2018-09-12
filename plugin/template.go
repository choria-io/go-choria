package plugin

const ftempl = `// auto generated {{.Now}}
package main

import (
	"github.com/choria-io/go-choria/plugin"
	p "{{.Repo}}"

)

func init() {
	err := plugin.Register("{{.Name}}", p.ChoriaPlugin())
	if err != nil {
		panic(err)
	}
}
`
