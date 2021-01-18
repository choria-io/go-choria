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
	"strings"
	"time"

	"github.com/choria-io/go-choria/client/client"
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

func (f *FlatFile) Discover(ctx context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{}

	for _, opt := range opts {
		opt(dopts)
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

	switch dopts.format {
	case TextFormat:
		return f.textDiscover(dopts.reader)

	case JSONFormat:
		return f.jsonDiscover(dopts.reader)

	case YAMLFormat:
		return f.yamlDiscover(dopts.reader)

	case ChoriaResponses:
		return f.choriaDiscover(dopts.reader)

	default:
		return nil, fmt.Errorf("unknow file format")
	}
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

func (f *FlatFile) yamlDiscover(file io.Reader) ([]string, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	nodes := []string{}
	err = yaml.Unmarshal(data, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (f *FlatFile) jsonDiscover(file io.Reader) ([]string, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	nodes := []string{}
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
