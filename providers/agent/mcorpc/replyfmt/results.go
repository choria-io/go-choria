package replyfmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/choria"
	rpc "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

type RPCStats struct {
	RequestID           string        `json:"requestid"`
	NoResponses         []string      `json:"no_responses"`
	UnexpectedResponses []string      `json:"unexpected_responses"`
	DiscoveredCount     int           `json:"discovered"`
	FailCount           int           `json:"failed"`
	OKCount             int           `json:"ok"`
	ResponseCount       int           `json:"responses"`
	PublishTime         time.Duration `json:"publish_time"`
	RequestTime         time.Duration `json:"request_time"`
	DiscoverTime        time.Duration `json:"discover_time"`
	StartTime           time.Time     `json:"start_time_utc"`
}

type RPCReply struct {
	Sender string `json:"sender"`
	*rpc.RPCReply
}

type RPCResults struct {
	Agent     string          `json:"agent"`
	Action    string          `json:"action"`
	Replies   []*RPCReply     `json:"replies"`
	Stats     *rpc.Stats      `json:"request_stats"`
	Summaries json.RawMessage `json:"summaries"`
}

type ActionDDL interface {
	SetOutputDefaults(results map[string]interface{})
	AggregateResult(result map[string]interface{}) error
	AggregateResultJSON(jres []byte) error
	AggregateSummaryJSON() ([]byte, error)
	GetOutput(string) (*agent.ActionOutputItem, bool)
	AggregateSummaryFormattedStrings() (map[string][]string, error)
	DisplayMode() string
}

type flusher interface {
	Flush()
}

func (r *RPCResults) RenderTXT(w io.Writer, action ActionDDL, verbose bool, silent bool, display DisplayMode, log *logrus.Entry) (err error) {
	fmtopts := []Option{
		Display(display),
	}

	if verbose {
		fmtopts = append(fmtopts, Verbose())
	}

	if silent {
		fmtopts = append(fmtopts, Silent())
	}

	for _, reply := range r.Replies {
		err := FormatReply(w, ConsoleFormat, action, reply.Sender, reply.RPCReply, fmtopts...)
		if err != nil {
			fmt.Fprintf(w, "Could not render reply from %s: %v", reply.Sender, err)
		}

		err = action.AggregateResultJSON(reply.Data)
		if err != nil {
			log.Warnf("could not aggregate data in reply: %v", err)
		}
	}

	if silent {
		return nil
	}

	FormatAggregates(w, ConsoleFormat, action, fmtopts...)

	fmt.Fprintln(w)

	stats := statsFromClient(r.Stats)

	if verbose {
		fmt.Fprintln(w, color.YellowString("---- request stats ----"))
		fmt.Fprintf(w, "               Nodes: %d / %d\n", stats.ResponseCount, stats.DiscoveredCount)
		fmt.Fprintf(w, "         Pass / Fail: %d / %d\n", stats.OKCount, stats.FailCount)
		fmt.Fprintf(w, "        No Responses: %d\n", len(stats.NoResponses))
		fmt.Fprintf(w, "Unexpected Responses: %d\n", len(stats.UnexpectedResponses))
		fmt.Fprintf(w, "          Start Time: %s\n", stats.StartTime.Format("2006-01-02T15:04:05-0700"))
		fmt.Fprintf(w, "      Discovery Time: %v\n", stats.DiscoverTime)
		fmt.Fprintf(w, "        Publish Time: %v\n", stats.PublishTime)
		fmt.Fprintf(w, "          Agent Time: %v\n", stats.RequestTime-stats.PublishTime)
		fmt.Fprintf(w, "          Total Time: %v\n", stats.RequestTime+stats.DiscoverTime)
	} else {
		fmt.Fprintf(w, "Finished processing %d / %d hosts in %s\n", stats.ResponseCount, stats.DiscoveredCount, stats.RequestTime+stats.DiscoverTime)
	}

	nodeListPrinter := func(nodes []string, message string) {
		if len(nodes) > 0 {
			sort.Strings(nodes)

			fmt.Fprintf(w, "\n%s: %d\n\n", message, len(nodes))

			out := bytes.NewBuffer([]byte{})

			w := new(tabwriter.Writer)
			w.Init(out, 0, 0, 4, ' ', 0)
			choria.SliceGroups(nodes, 3, func(g []string) {
				fmt.Fprintln(w, "    "+strings.Join(g, "\t")+"\t")
			})
			w.Flush()

			fmt.Fprint(w, out.String())
		}
	}

	nodeListPrinter(stats.NoResponses, "No Responses from")
	nodeListPrinter(stats.UnexpectedResponses, "Unexpected Responses from")

	f, ok := w.(flusher)
	if ok {
		f.Flush()
	}

	return nil
}

func (r *RPCResults) RenderJSON(w io.Writer, action ActionDDL) (err error) {
	for _, reply := range r.Replies {
		parsed, ok := gjson.ParseBytes(reply.RPCReply.Data).Value().(map[string]interface{})
		if ok {
			action.SetOutputDefaults(parsed)
			action.AggregateResult(parsed)
		}
	}

	// silently failing as this is optional
	r.Summaries, _ = action.AggregateSummaryJSON()

	j, err := json.MarshalIndent(r, "", "   ")
	if err != nil {
		return fmt.Errorf("could not prepare display: %s", err)
	}

	_, err = fmt.Fprintln(w, string(j))

	return err
}

func statsFromClient(cs *rpc.Stats) *RPCStats {
	s := &RPCStats{}

	s.RequestID = cs.RequestID
	s.NoResponses = cs.NoResponseFrom()
	s.UnexpectedResponses = cs.UnexpectedResponseFrom()
	s.DiscoveredCount = cs.DiscoveredCount()
	s.FailCount = cs.FailCount()
	s.OKCount = cs.OKCount()
	s.ResponseCount = cs.ResponsesCount()
	s.StartTime = cs.Started().UTC()

	d, err := cs.DiscoveryDuration()
	if err == nil {
		s.DiscoverTime = d
	}

	d, err = cs.PublishDuration()
	if err == nil {
		s.PublishTime = d
	}

	d, err = cs.RequestDuration()
	if err == nil {
		s.RequestTime = d
	}

	return s
}
