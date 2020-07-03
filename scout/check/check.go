package check

import (
	"time"
)

type Check struct {
	Name              string        `json:"name"`
	Plugin            string        `json:"plugin"`
	Builtin           string        `json:"builtin"`
	Arguments         string        `json:"arguments"`
	PluginTimeout     time.Duration `json:"plugin_timeout"`
	CheckInterval     time.Duration `json:"check_interval"`
	RemediateCommand  string        `json:"remediate_command"`
	RemediateInterval time.Duration `json:"remediate_interval"`
}
