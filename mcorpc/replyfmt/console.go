package replyfmt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"

	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/client"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc/ddl/agent"
)

type ConsoleFormatter struct {
	verbose         bool
	silent          bool
	disableColor    bool
	displayOverride DisplayMode

	actionInterface *agent.Action
	out             *bufio.Writer
}

type statusString struct {
	color string
	plain string
}

var statusStings = map[mcorpc.StatusCode]statusString{
	mcorpc.OK:            {"", ""},
	mcorpc.Aborted:       {color.RedString("Request Aborted"), "Request Aborted"},
	mcorpc.InvalidData:   {color.YellowString("Invalid Request Data"), "Invalid Request Data"},
	mcorpc.MissingData:   {color.YellowString("Missing Request Data"), "Missing Request Data"},
	mcorpc.UnknownAction: {color.YellowString("Unknown Action"), "Unknown Action"},
	mcorpc.UnknownError:  {color.RedString("Unknown Request Status"), "Unknown Request Status"},
}

func NewConsoleFormatter(opts ...Option) *ConsoleFormatter {
	f := &ConsoleFormatter{}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// ConsoleNoColor disables color in the console formatter
func ConsoleNoColor() Option {
	return func(f Formatter) error {
		i, ok := f.(*ConsoleFormatter)
		if !ok {
			return fmt.Errorf("formatter is not a ConsoleFormatter")
		}

		i.disableColor = true

		return nil
	}
}

func (c *ConsoleFormatter) Format(w *bufio.Writer, action *agent.Action, sender string, reply *client.RPCReply) error {
	c.out = w
	c.actionInterface = action

	if !c.shouldDisplayReply(reply) {
		return nil
	}

	defer w.Flush()

	c.writeHeader(sender, reply)

	if c.verbose {
		c.basicPrinter(reply)
		return nil
	}

	if reply.Statuscode > mcorpc.OK {
		c.errorPrinter(reply)
		return nil
	}

	c.ddlAssistedPrinter(reply)

	return nil
}

func (c *ConsoleFormatter) SetVerbose() {
	c.verbose = true
}

func (c *ConsoleFormatter) SetSilent() {
	c.silent = true
}

func (c *ConsoleFormatter) SetDisplay(m DisplayMode) {
	c.displayOverride = m
}

func (c *ConsoleFormatter) errorPrinter(reply *client.RPCReply) {
	if c.disableColor {
		fmt.Fprintf(c.out, "    %s\n", reply.Statusmsg)
	} else {
		fmt.Fprintf(c.out, "    %s\n", color.YellowString(reply.Statusmsg))
	}

	fmt.Fprintln(c.out)
}

func (c *ConsoleFormatter) writeHeader(sender string, reply *client.RPCReply) {
	ss := statusStings[reply.Statuscode]
	smsg := "%-40s %s\n"
	if c.disableColor {
		fmt.Fprintf(c.out, smsg, sender, ss.color)
	} else {
		fmt.Fprintf(c.out, smsg, sender, ss.plain)
	}
}

func (c *ConsoleFormatter) ddlAssistedPrinter(reply *client.RPCReply) {
	max := 0
	keys := []string{}

	parsed, ok := gjson.ParseBytes(reply.Data).Value().(map[string]interface{})
	if ok {
		c.actionInterface.SetOutputDefaults(parsed)
		c.actionInterface.AggregateResult(parsed)
	}

	for key := range parsed {
		output, ok := c.actionInterface.Output[key]
		if ok {
			if len(output.DisplayAs) > max {
				max = len(output.DisplayAs)
			}
		} else {
			if len(key) > max {
				max = len(key)
			}
		}

		keys = append(keys, key)
	}

	formatStr := fmt.Sprintf("%%%ds: %%s\n", max+3)
	prefixFormatStr := fmt.Sprintf("%%%ds", max+5)

	sort.Strings(keys)

	for _, key := range keys {
		val := gjson.GetBytes(reply.Data, key)
		keyStr := key
		valStr := val.String()

		output, ok := c.actionInterface.Output[key]
		if ok {
			keyStr = output.DisplayAs
		}

		if val.IsArray() || val.IsObject() {
			valStr = string(pretty.PrettyOptions([]byte(valStr), &pretty.Options{
				SortKeys: true,
				Prefix:   fmt.Sprintf(prefixFormatStr, " "),
				Indent:   "   ",
				Width:    80,
			}))
		}

		fmt.Fprintf(c.out, formatStr, keyStr, strings.TrimLeft(valStr, " "))
	}

	fmt.Fprintln(c.out)
}

func (c *ConsoleFormatter) basicPrinter(reply *client.RPCReply) {
	j, err := json.MarshalIndent(reply.Data, "   ", "   ")
	if err != nil {
		fmt.Fprintf(c.out, "   %s\n", string(reply.Data))
	}

	fmt.Fprintf(c.out, "   %s\n", string(j))
}

func (c *ConsoleFormatter) shouldDisplayReply(reply *client.RPCReply) bool {
	switch c.displayOverride {
	case DisplayDDL:
		if reply.Statuscode > mcorpc.OK && c.actionInterface.Display == "failed" {
			return true
		} else if reply.Statuscode > mcorpc.OK && c.actionInterface.Display == "" {
			return true
		} else if c.actionInterface.Display == "ok" && reply.Statuscode == mcorpc.OK {
			return true
		} else if c.actionInterface.Display == "always" {
			return true
		}
	case DisplayOK:
		if reply.Statuscode == mcorpc.OK {
			return true
		}
	case DisplayFailed:
		if reply.Statuscode > mcorpc.OK {
			return true
		}
	case DisplayAll:
		return true
	case DisplayNone:
		return false
	}

	return false
}
