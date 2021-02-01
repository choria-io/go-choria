package scoutcmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/client/discovery"
	"github.com/choria-io/go-choria/client/scoutclient"
	"github.com/choria-io/go-choria/internal/util"
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
	return util.NewMarkdownTable(hdr...)
}
