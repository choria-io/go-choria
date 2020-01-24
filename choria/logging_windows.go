package choria

import (
	"strings"

	"github.com/Freman/eventloghook"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func (fw *Framework) openLogAndLogErr(lerr error) error {
	err := fw.commonLogOpener()
	if err != nil {
		return err
	}

	fw.log.Errorf("could not log to event log: %s", err)

	return nil
}

func (fw *Framework) openLogfile() error {
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		// if this failed we always log to file
		interactive = false
	}

	// if its the server and we are not interactive we log to event log
	if fw.Config.InitiatedByServer && !interactive {
		err := eventlog.InstallAsEventCreate("choria-server", eventlog.Error|eventlog.Warning|eventlog.Info)
		if err != nil && !strings.Contains(err.Error(), "already exist") {
			return fw.openLogAndLogErr(err)
		}

		elog, err := eventlog.Open("choria-server")
		if err != nil {
			return fw.openLogAndLogErr(err)
		}

		log.AddHook(eventloghook.NewHook(elog))
		fw.log.AddHook(eventloghook.NewHook(elog))

		return nil
	}

	// if its not the server we log to whatever is configured
	return fw.commonLogOpener()
}
