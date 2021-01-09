package client

import (
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"

	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

func (g *Generator) templFSnakeToCamel(v string) string {
	parts := strings.Split(v, "_")
	out := []string{}
	for _, s := range parts {
		out = append(out, strings.Title(s))
	}

	return strings.Join(out, "")
}

func (g *Generator) templFChoriaTypeToValOfType(v string) string {
	switch v {
	case "string", "list":
		return "val.(string)"
	case "integer":
		return "val.(int64)"
	case "number", "float":
		return "val.(float64)"
	case "boolean":
		return "val.(bool)"
	case "hash":
		return "val.(map[string]interface{})"
	case "array":
		return "val.([]interface{})"
	default:
		return "val.(interface{})"
	}
}

func (g *Generator) templFChoriaRequiredInputsToFuncArgs(act *addl.Action) string {
	inputs := g.optionalInputSelect(act, false)
	parts := []string{}

	for name, input := range inputs {
		goType := g.templFChoriaTypeToGo(input.Type)
		parts = append(parts, fmt.Sprintf("%sI %s", strings.ToLower(name), goType))
	}

	return strings.Join(parts, ", ")
}

func (g *Generator) templFChoriaTypeToGo(v string) string {
	switch v {
	case "string", "list":
		return "string"
	case "integer":
		return "int64"
	case "number", "float":
		return "float64"
	case "boolean":
		return "bool"
	case "hash":
		return "map[string]interface{}"
	case "array":
		return "[]interface{}"
	default:
		return "interface{}"
	}
}

func (g *Generator) templFChoriaOptionalInputs(act *addl.Action) map[string]*common.InputItem {
	return g.optionalInputSelect(act, true)
}

func (g *Generator) templFChoriaRequiredInputs(act *addl.Action) map[string]*common.InputItem {
	return g.optionalInputSelect(act, false)
}

func (g *Generator) templFBase64Encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func (g *Generator) templFGeneratedWarning() string {
	meta := g.agent.DDL.Metadata
	return fmt.Sprintf(`// generated code; DO NOT EDIT; %v"
//
// Client for Choria RPC Agent '%s'' Version %s generated using Choria version %s`, time.Now(), meta.Name, meta.Version, choria.BuildInfo().Version())
}

func (g *Generator) funcMap() template.FuncMap {
	return template.FuncMap{
		"GeneratedWarning":               g.templFGeneratedWarning,
		"Base64Encode":                   g.templFBase64Encode,
		"Capitalize":                     strings.Title,
		"ToLower":                        strings.ToLower,
		"SnakeToCamel":                   g.templFSnakeToCamel,
		"ChoriaRequiredInputs":           g.templFChoriaRequiredInputs,
		"ChoriaOptionalInputs":           g.templFChoriaOptionalInputs,
		"ChoriaRequiredInputsToFuncArgs": g.templFChoriaRequiredInputsToFuncArgs,
		"ChoriaTypeToGoType":             g.templFChoriaTypeToGo,
		"ChoriaTypeToValOfType":          g.templFChoriaTypeToValOfType,
	}
}
