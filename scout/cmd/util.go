package scoutcmd

import (
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/scoutclient"
)

type StandardOptions struct {
	Collective string
	FactF      []string
	ClassF     []string
	IdentityF  []string
	CombinedF  []string
	CompoundF  string
	DM         string
	DT         int
}

func (s *StandardOptions) scoutClient(cfile string, log *logrus.Entry) (*scoutclient.ScoutClient, error) {
	co := []scoutclient.InitializationOption{
		scoutclient.ConfigFile(cfile), scoutclient.Logger(log), scoutclient.Progress(),
	}

	switch s.DM {
	case "choria", "puppetdb":
		co = append(co, scoutclient.Discovery(&scoutclient.PuppetDBNS{}))
	default:
		co = append(co, scoutclient.Discovery(&scoutclient.BroadcastNS{}))
	}

	sc, err := scoutclient.New(co...)
	if err != nil {
		return nil, err
	}

	sc.OptionIdentityFilter(s.IdentityF...)
	sc.OptionClassFilter(s.ClassF...)
	sc.OptionFactFilter(s.FactF...)
	sc.OptionCombinedFilter(s.CombinedF...)
	sc.OptionCollective(s.Collective)
	sc.OptionCompoundFilter(s.CompoundF)
	sc.OptionDiscoveryTimeout(time.Duration(s.DT) * time.Second)

	return sc, nil
}

func newMarkdownTable(hdr ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader(hdr)

	return table
}
