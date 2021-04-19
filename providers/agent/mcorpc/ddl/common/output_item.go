package common

import "github.com/choria-io/go-choria/internal/fs"

var (
	OutputTypeArray   = "Array"
	OutputTypeBoolean = "boolean"
	OutputTypeFloat   = "float"
	OutputTypeHash    = "Hash"
	OutputTypeInteger = "integer"
	OutputTypeList    = "list"
	OutputTypeNumber  = "number"
	OutputTypeString  = "string"
	OutputTypeAny     = ""
)

// OutputItem describes an individual output item
type OutputItem struct {
	Description string      `json:"description"`
	DisplayAs   string      `json:"display_as"`
	Default     interface{} `json:"default,omitempty"`
	Type        string      `json:"type,omitempty"`
}

func (i *OutputItem) RenderConsole() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/console/output_item.templ", i, nil)
}

func (i *OutputItem) RenderMarkdown() ([]byte, error) {
	return fs.ExecuteTemplate("ddl/markdown/output_item.templ", i, nil)
}
