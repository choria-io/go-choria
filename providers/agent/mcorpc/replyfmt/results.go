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
	"github.com/olekukonko/tablewriter"
	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
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
	Agent       string          `json:"agent"`
	Action      string          `json:"action"`
	Replies     []*RPCReply     `json:"replies"`
	Stats       *rpc.Stats      `json:"-"`
	ParsedStats *RPCStats       `json:"request_stats"`
	Summaries   json.RawMessage `json:"summaries"`
}

type ActionDDL interface {
	SetOutputDefaults(results map[string]interface{})
	AggregateResult(result map[string]interface{}) error
	AggregateResultJSON(jres []byte) error
	AggregateSummaryJSON() ([]byte, error)
	GetOutput(string) (*agent.ActionOutputItem, bool)
	AggregateSummaryFormattedStrings() (map[string][]string, error)
	DisplayMode() string
	OutputNames() []string
}

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
}

type flusher interface {
	Flush()
}

func (r *RPCResults) RenderTXTFooter(w io.Writer, verbose bool) {
	stats := statsFromClient(r.Stats)

	if verbose {
		fmt.Fprintln(w, color.YellowString("---- request stats ----"))
		fmt.Fprintf(w, "               Nodes: %d / %d\n", stats.ResponseCount, stats.DiscoveredCount)
		fmt.Fprintf(w, "         Pass / Fail: %d / %d\n", stats.OKCount, stats.FailCount)
		fmt.Fprintf(w, "        No Responses: %d\n", len(stats.NoResponses))
		fmt.Fprintf(w, "Unexpected Responses: %d\n", len(stats.UnexpectedResponses))
		fmt.Fprintf(w, "          Start Time: %s\n", stats.StartTime.Format("2006-01-02T15:04:05-0700"))
		fmt.Fprintf(w, "      Discovery Time: %v\n", stats.DiscoverTime.Round(time.Millisecond))
		fmt.Fprintf(w, "        Publish Time: %v\n", stats.PublishTime.Round(time.Millisecond))
		fmt.Fprintf(w, "          Agent Time: %v\n", (stats.RequestTime - stats.PublishTime).Round(time.Millisecond))
		fmt.Fprintf(w, "          Total Time: %v\n", (stats.RequestTime + stats.DiscoverTime).Round(time.Millisecond))
	} else {
		fmt.Fprintf(w, "Finished processing %d / %d hosts in %s\n", stats.ResponseCount, stats.DiscoveredCount, (stats.RequestTime + stats.DiscoverTime).Round(time.Millisecond))
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
}

func (r *RPCResults) RenderTXT(w io.Writer, action ActionDDL, verbose bool, silent bool, display DisplayMode, log Logger) (err error) {
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

	r.RenderTXTFooter(w, verbose)

	f, ok := w.(flusher)
	if ok {
		f.Flush()
	}

	return nil
}

// RenderTable renders a table of outputs
// TODO: should become a reply format formatter, but those lack a prepare phase to print headers etc
func (r *RPCResults) RenderTable(w io.Writer, action ActionDDL) (err error) {
	table := tablewriter.NewWriter(w)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	var (
		headers = []string{"sender"}
		outputs = action.OutputNames()
	)

	for _, o := range outputs {
		output, ok := action.GetOutput(o)
		if ok {
			headers = append(headers, output.DisplayAs)
		} else {
			headers = append(headers, strings.Title(o))
		}
	}

	if len(headers) == 0 {
		return nil
	}

	table.SetHeader(headers)

	for _, reply := range r.Replies {
		if reply.Statuscode != mcorpc.OK {
			continue
		}

		parsedResult := gjson.ParseBytes(reply.RPCReply.Data)
		if parsedResult.Exists() {
			row := []string{reply.Sender}
			for _, o := range outputs {
				val := parsedResult.Get(o)
				switch {
				case val.IsArray(), val.IsObject():
					row = append(row, string(pretty.PrettyOptions([]byte(val.String()), &pretty.Options{
						SortKeys: true,
					})))
				default:
					row = append(row, val.String())
				}
			}
			table.Append(row)
		}
	}

	table.Render()

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
	r.ParsedStats = statsFromClient(r.Stats)

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
