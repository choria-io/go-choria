package cmd

import (
	"fmt"
	"net/http"

	"github.com/choria-io/go-lifecycle/tally"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func recordtally() error {
	log := fw.Logger("tally")
	log.Infof("Choria Lifecycle Tally version %s starting listening on port %d", Version, port)

	conn, err := fw.NewConnector(ctx, fw.MiddlewareServers, fw.Certname(), log)
	if err != nil {
		return errors.Wrap(err, "cannot connect")
	}

	recorder, err := tally.New(tally.Component(componentFilter), tally.Logger(log), tally.StatsPrefix(prefix), tally.Connection(conn))
	if err != nil {
		return errors.Wrap(err, "could not create recorder")
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	err = recorder.Run(ctx)
	if err != nil {
		return errors.Wrap(err, "recorder failed")
	}

	return nil
}
