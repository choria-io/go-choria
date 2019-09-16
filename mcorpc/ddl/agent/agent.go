package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

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

// Find looks in the supplied libdirs for a DDL file for a specific agent
func Find(agent string, libdirs []string) (ddl *DDL, err error) {
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
	for _, dir := range libdirs {
		agentsdir := filepath.Join(dir, "mcollective", "agent")

		filepath.Walk(agentsdir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			_, name := filepath.Split(path)
			extension := filepath.Ext(name)

			if extension != ".json" {
				return nil
			}

			cb(strings.TrimSuffix(name, extension), path)

			return nil
		})
	}
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
	if err != nil {
		return false
	}

	return true
}

// Timeout is the timeout for this agent
func (d *DDL) Timeout() time.Duration {
	if d.Metadata.Timeout == 0 {
		return time.Duration(10 * time.Second)
	}

	return time.Duration(time.Second * time.Duration(d.Metadata.Timeout))
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
				return fmt.Sprintf("%f", v.(float64))
			case "integer":
				if v == nil {
					return "0"
				}
				return fmt.Sprintf("%d", v.(int64))
			case "boolean":
				if v == nil {
					return "false"
				}
				return fmt.Sprintf("%v", v.(bool))
			}

			return `nil`
		},
	}

	tpl := template.Must(template.New(d.Metadata.Name).Funcs(funcs).Parse(rubyDDLTemplate))
	err := tpl.Execute(&out, d)
	return out.String(), err
}

var rubyDDLTemplate = `metadata :name        => "{{ .Metadata.Name }}",
         :description => "{{ .Metadata.Description }}",
         :author      => "{{ .Metadata.Author }}",
         :license     => "{{ .Metadata.License }}",
         :version     => "{{ .Metadata.Version }}",
         :url         => "{{ .Metadata.URL }}"
         :timeout     => {{ .Metadata.Timeout }}

{{ range $aname, $action := .Actions }}
action "{{ $action.Name }}", :description => "{{ $action.Description }}" do
  display :{{ $action.Display }}
{{ range $iname, $input := $action.Input }}
  input :{{ $iname }},
        :prompt      => "{{ $input.Prompt }}",
        :description => "{{ $input.Description }}",
        :type        => :{{ $input.Type }},
        :optional    => {{ $input.Optional }},
{{- if $input.Default }}
        :default     => {{ $input.Default | goval2rubyval $input.Type }}
{{- end -}}
{{- if eq $input.Type "string" }}
        :validation  => :{{ $input.Validation }},
        :maxlength   => {{ $input.MaxLength }},
{{- end -}}
{{- if eq $input.Type "list" }}
        :list        => {{ $input.Enum | enum2list }}
{{- end -}}

{{ end }}

{{ range $oname, $output := $action.Output }}
  output :{{ $oname }},
         :description => "{{ $output.Description }}",
         :display_as  => "{{ $output.DisplayAs }}",
         :type        => "{{ $output.Type }}",
{{- if $output.Default }}
         :default     => {{ $output.Default | goval2rubyval $output.Type }}
{{- end -}}
{{ end }}

{{- if $action.Aggregation }}
  summarize do
{{- range $aname, $aggregate := $action.Aggregation }}
    {{ $aggregate.Function }}(:{{ $aggregate.OutputName }})
{{- end }}
  end
{{- end }}
end
{{ end }}
`
