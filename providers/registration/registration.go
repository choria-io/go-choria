package registration

import (
	"bytes"
	"compress/gzip"
	"encoding/json"

	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/go-choria/statistics"
)

type ServerInfoSource interface {
	Classes() []string
	Facts() json.RawMessage
	Identity() string
	KnownAgents() []string
	DataFuncMap() (ddl.FuncMap, error)
	Status() *statistics.InstanceStatus
	AgentMetadata(agent string) (agents.Metadata, bool)
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	gz := gzip.NewWriter(&b)

	_, err := gz.Write(data)
	if err != nil {
		return []byte{}, err
	}

	err = gz.Flush()
	if err != nil {
		return []byte{}, err
	}

	err = gz.Close()
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}
