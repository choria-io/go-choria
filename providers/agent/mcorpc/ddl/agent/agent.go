package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/choria-io/go-choria/internal/templates"
	common "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/common"
	"github.com/choria-io/go-choria/server/agents"
)

// DDL represents the schema of a mcorpc agent and is at a basic level
// compatible with the mcollective agent ddl format
type DDL struct {
	Schema         string           `json:"$schema"`
	Metadata       *agents.Metadata `json:"metadata"`
	Actions        []*Action        `json:"actions"`
	SourceLocation string           `json:"-"`
}

// New creates a new DDL from a JSON file
func New(file string) (*DDL, error) {
	ddl := &DDL{
		SourceLocation: file,
	}

	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not load DDL data: %s", err)
	}

	err = json.Unmarshal(dat, ddl)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON data in %s: %s", file, err)
	}

	ddl.normalize()

	return ddl, nil
}

// FindAll loads all plugins from the libdirs and optionally the cache
func FindAll(libdirs []string, cached bool) ([]*DDL, error) {
	var found []*DDL
	var err error
	var ddl *DDL

	EachFile(libdirs, func(n string, f string) bool {
		ddl, err = New(f)
		if err != nil {
			return false
		}

		found = append(found, ddl)

		return true
	})
	if err != nil {
		return nil, err
	}

	if cached {
		for _, name := range CachedDDLs() {
			ddl, err = CachedDDL(name)
			if err != nil {
				return nil, err
			}
			found = append(found, ddl)
		}
	}

	return found, nil
}

// Find looks in the supplied libdirs for a DDL file for a specific agent
func Find(agent string, libdirs []string) (ddl *DDL, err error) {
	ddl, err = CachedDDL(agent)
	if err == nil {
		return ddl, nil
	}

	EachFile(libdirs, func(n string, f string) bool {
		if n == agent {
			ddl, err = New(f)
			return true
		}

		return false
	})

	if err != nil {
		return nil, fmt.Errorf("could not load agent %s: %s", agent, err)
	}

	if ddl == nil {
		return nil, fmt.Errorf("could not find DDL file for %s", agent)
	}

	return ddl, nil
}

// EachFile calls cb with a path to every found agent DDL, stops looking when br is true
func EachFile(libdirs []string, cb func(name string, path string) (br bool)) {
	common.EachFile("agent", libdirs, cb)
}

func (d *DDL) normalize() {
	for _, action := range d.Actions {
		if action.Display == "" {
			action.Display = "failed"
		}
	}
}

// ActionNames is a list of known actions defined by a DDL
func (d *DDL) ActionNames() []string {
	actions := []string{}

	if d.Actions != nil {
		for _, act := range d.Actions {
			actions = append(actions, act.Name)
		}
	}

	sort.Strings(actions)

	return actions
}

// ActionInterface looks up an Action by name
func (d *DDL) ActionInterface(action string) (*Action, error) {
	for _, act := range d.Actions {
		if act.Name == action {
			return act, nil
		}
	}

	return nil, fmt.Errorf("unknown action %s#%s", d.Metadata.Name, action)
}

// HaveAction determines if an action is known
func (d *DDL) HaveAction(action string) bool {
	_, err := d.ActionInterface(action)
	return err == nil
}

// Timeout is the timeout for this agent
func (d *DDL) Timeout() time.Duration {
	if d.Metadata.Timeout == 0 {
		return 10 * time.Second
	}

	return time.Second * time.Duration(d.Metadata.Timeout)
}

// ValidateAndConvertToDDLTypes converts args to the correct data types as declared in the DDL and validates everything
func (d *DDL) ValidateAndConvertToDDLTypes(action string, args map[string]string) (result map[string]interface{}, warnings []string, err error) {
	acti, err := d.ActionInterface(action)
	if err != nil {
		return result, warnings, err
	}

	return acti.ValidateAndConvertToDDLTypes(args)
}

// ToRuby generates a ruby DDL from a go DDL
func (d *DDL) ToRuby() (string, error) {
	var out bytes.Buffer

	funcs := template.FuncMap{
		"fmtAggregateArguments": func(output string, v json.RawMessage) string {
			var args []interface{}
			err := json.Unmarshal(v, &args)
			if err != nil {
				return fmt.Sprintf(":%s", output)
			}

			switch len(args) {
			case 1:
				return fmt.Sprintf(":%v", args[0])
			case 2:
				opts := ""
				margs, ok := args[1].(map[string]interface{})
				if ok {
					for k, v := range margs {
						vs, ok := v.(string)
						if ok {
							opts = fmt.Sprintf(":%s => %q", k, vs)
						}
					}
					return fmt.Sprintf(":%v, %s", args[0], opts)
				}

				return fmt.Sprintf(":%v", args[0])

			default:
				return fmt.Sprintf(":%s", output)
			}
		},
		"validatorStr": func(v string) string {
			if v == "" {
				return `"."`
			}

			switch v {
			case "shellsafe", "ipv4address", "ipv6address", "ipaddress":
				return ":" + v
			default:
				return `'` + v + `'`
			}
		},

		"enum2list": func(v []string) string {
			if len(v) == 0 {
				return "[]"
			}

			return `["` + strings.Join(v, `", "`) + `"]`
		},

		"goval2rubyval": func(typedef string, v interface{}) string {
			if v == nil {
				return `nil`
			}

			switch typedef {
			case "string", "list":
				if v == nil {
					return `""`
				}

				return fmt.Sprintf(`"%s"`, v.(string))
			case "float", "number":
				if v == nil {
					return "0.0"
				}

				switch val := reflect.ValueOf(v); val.Kind() {
				case reflect.Int:
					return fmt.Sprintf("%d", v.(int))

				case reflect.Int16:
					return fmt.Sprintf("%d", v.(int16))

				case reflect.Int32:
					return fmt.Sprintf("%d", v.(int32))

				case reflect.Int64:
					return fmt.Sprintf("%d", v.(int64))

				case reflect.Float32:
					return fmt.Sprintf("%f", math.Round(float64(v.(float32))))

				case reflect.Float64:
					return fmt.Sprintf("%f", math.Round(v.(float64)))

				default:
					panic(fmt.Sprintf("unknown value type %v found in %s field", val.Kind(), typedef))
				}

			case "integer":
				if v == nil {
					return "0"
				}

				switch val := reflect.ValueOf(v); val.Kind() {
				case reflect.Int:
					return fmt.Sprintf("%d", v.(int))

				case reflect.Int16:
					return fmt.Sprintf("%d", v.(int16))

				case reflect.Int32:
					return fmt.Sprintf("%d", v.(int32))

				case reflect.Int64:
					return fmt.Sprintf("%d", v.(int64))

				case reflect.Float32:
					return fmt.Sprintf("%.0f", math.Round(float64(v.(float32))))

				case reflect.Float64:
					return fmt.Sprintf("%.0f", math.Round(v.(float64)))

				default:
					panic(fmt.Sprintf("unknown value type %v found in %s field", val.Kind(), typedef))
				}

			case "boolean":
				if v == nil {
					return "false"
				}

				return fmt.Sprintf("%v", v.(bool))
			}

			return `nil`
		},
	}

	rubyDDLTemplate, err := templates.FS.ReadFile("ddl/agent_ruby_ddl.templ")
	if err != nil {
		return "", err
	}

	tpl := template.Must(template.New(d.Metadata.Name).Funcs(funcs).Parse(string(rubyDDLTemplate)))
	err = tpl.Execute(&out, d)
	return out.String(), err
}
