package ddl

import (
	"github.com/choria-io/go-choria/internal/templates"
)

// RenderConsole create console appropriate output for data provider ddls
func (d *DDL) RenderConsole() ([]byte, error) {
	return templates.ExecuteTemplate("ddl/data_provider_console.templ", d, nil)
}
