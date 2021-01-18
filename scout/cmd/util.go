package scoutcmd

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/scoutclient"
)

func scoutClient(cfile string, opt *discovery.StandardOptions, log *logrus.Entry) (*scoutclient.ScoutClient, error) {
	co := []scoutclient.InitializationOption{
		scoutclient.ConfigFile(cfile),
		scoutclient.Logger(log),
		scoutclient.Progress(),
		scoutclient.Discovery(&scoutclient.MetaNS{
			Options:               opt,
			Agent:                 "scout",
			DisablePipedDiscovery: false,
		}),
	}

	sc, err := scoutclient.New(co...)
	if err != nil {
		return nil, err
	}

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
