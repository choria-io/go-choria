package agent

import (
	"github.com/choria-io/go-choria/internal/fs"
)

// RenderConsole create console appropriate output for agent provider ddls
func (d *DDL) RenderConsole() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/console/agent.templ", d, nil)
}

// RenderMarkdown create markdown appropriate output for agent provider ddls
func (d *DDL) RenderMarkdown() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/markdown/agent.templ", d, nil)
}
