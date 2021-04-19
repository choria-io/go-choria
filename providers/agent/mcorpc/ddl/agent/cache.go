package agent

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/choria-io/go-choria/internal/fs"
)

func CachedDDLs() []string {
	dir, err := fs.FS.ReadDir("ddl/cache/agent")
	if err != nil {
		return nil
	}

	names := []string{}
	for _, f := range dir {
		if f.IsDir() {
			continue
		}

		ext := filepath.Ext(f.Name())
		if ext != ".json" {
			continue
		}

		names = append(names, strings.TrimSuffix(f.Name(), ext))
	}

	return names
}

// CachedDDLBytes is the raw JSON encoded DDL file for the agent
func CachedDDLBytes(agent string) ([]byte, error) {
	return fs.FS.ReadFile("ddl/cache/agent/" + agent + ".json")
}

// CachedDDL is a parsed and loaded DDL for the agent a
func CachedDDL(a string) (*DDL, error) {
	ddlj, err := CachedDDLBytes(a)
	if err != nil {
		return nil, err
	}

	ddl := &DDL{}
	err = json.Unmarshal(ddlj, ddl)
	if err != nil {
		return nil, err
	}

	return ddl, nil
}
