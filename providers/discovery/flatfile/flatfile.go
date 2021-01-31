package flatfile

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/filter/identity"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/replyfmt"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

type FlatFile struct {
	fw      client.ChoriaFramework
	timeout time.Duration
	log     *logrus.Entry
}

func New(fw client.ChoriaFramework) *FlatFile {
	return &FlatFile{
		fw:      fw,
		timeout: time.Second * time.Duration(fw.Configuration().DiscoveryTimeout),
		log:     fw.Logger("flatfile_discovery"),
	}
}

func (f *FlatFile) Discover(_ context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{do: make(map[string]string)}

	for _, opt := range opts {
		opt(dopts)
	}

	if dopts.filter != nil {
		if len(dopts.filter.Agent) > 0 || len(dopts.filter.Compound) > 0 || len(dopts.filter.Class) > 0 || len(dopts.filter.Fact) > 0 {
			return nil, fmt.Errorf("only identity filters are supported")
		}
	}

	file, ok := dopts.do["file"]
	if ok {
		dopts.reader = nil
		dopts.source = file
	}

	format, ok := dopts.do["format"]
	if ok {
		switch format {
		case "json":
			dopts.format = JSONFormat
		case "yaml", "yml":
			dopts.format = YAMLFormat
		case "choriarpc", "results", "rpc", "response":
			dopts.format = ChoriaResponsesFormat
		default:
			dopts.format = TextFormat
		}
	}

	if dopts.source == "" && dopts.reader == nil {
		return nil, fmt.Errorf("source file not specified")
	}

	if dopts.reader == nil {
		sf, err := os.Open(dopts.source)
		if err != nil {
			return nil, err
		}
		defer sf.Close()

		dopts.reader = sf
	}

	var nodes []string

	switch dopts.format {
	case TextFormat, unknownFormat:
		nodes, err = f.textDiscover(dopts.reader)

	case JSONFormat:
		nodes, err = f.jsonDiscover(dopts.reader, dopts.do)

	case YAMLFormat:
		nodes, err = f.yamlDiscover(dopts.reader, dopts.do)

	case ChoriaResponsesFormat:
		nodes, err = f.choriaDiscover(dopts.reader)

	default:
		return nil, fmt.Errorf("unknown file format")
	}

	if err != nil {
		return nil, err
	}

	err = f.validateNodes(nodes)
	if err != nil {
		return nil, err
	}

	if dopts.filter != nil && len(dopts.filter.Identity) > 0 {
		matched := []string{}
		for _, idf := range dopts.filter.Identity {
			matched = append(matched, identity.FilterNodes(nodes, idf)...)
		}
		return matched, nil
	}

	return nodes, nil
}

func (f *FlatFile) validateNodes(nodes []string) error {
	matcher, err := regexp.Compile(`^(([a-zA-Z]|[a-zA-Z][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z]|[A-Za-z][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		if !matcher.MatchString(n) {
			return fmt.Errorf("invalid identity string %q", n)
		}
	}

	return nil
}

func (f *FlatFile) choriaDiscover(file io.Reader) ([]string, error) {
	raw, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	input := bytes.TrimSpace(raw)
	if len(input) < 2 {
		return nil, fmt.Errorf("did not detect valid JSON data")
	}

	if input[0] != '{' && input[len(input)-1] != '}' {
		return nil, fmt.Errorf("did not detect valid JSON data")
	}

	data := replyfmt.RPCResults{}
	err = json.Unmarshal(input, &data)
	if err != nil {
		return nil, err
	}

	found := []string{}
	for _, reply := range data.Replies {
		found = append(found, reply.Sender)
	}

	return found, nil
}

func (f *FlatFile) yamlDiscover(file io.Reader, do map[string]string) ([]string, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	jdata, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}

	return f.jsonDiscover(bytes.NewReader(jdata), do)
}

func (f *FlatFile) jsonDiscover(file io.Reader, do map[string]string) ([]string, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	nodes := []string{}
	filter, ok := do["filter"]
	if ok {
		if filter == "" {
			return nil, fmt.Errorf("empty filter string found in discovery options")
		}

		res := gjson.GetBytes(data, filter)
		if res.IsArray() {
			res.ForEach(func(_ gjson.Result, v gjson.Result) bool {
				if v.Exists() && v.Type == gjson.String {
					nodes = append(nodes, v.String())
				}

				return true
			})
			return nodes, nil
		} else {
			return nodes, fmt.Errorf("query %q did not result in a array of nodes", filter)
		}
	}

	err = json.Unmarshal(data, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (f *FlatFile) textDiscover(file io.Reader) ([]string, error) {
	var found []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		found = append(found, strings.TrimSpace(scanner.Text()))
	}

	err := scanner.Err()
	if err != nil {
		return nil, err
	}

	return found, nil
}
