package ddl

import "github.com/choria-io/go-choria/internal/fs"

// RenderConsole create console appropriate output for data provider ddls
func (d *DDL) RenderConsole() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/console/data_provider.templ", d, nil)
}

// RenderMarkdown create markdown appropriate output for data provider ddls
func (d *DDL) RenderMarkdown() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/markdown/data_provider.templ", d, nil)
}
