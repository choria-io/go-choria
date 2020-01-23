package choria

import (
	"github.com/Freman/eventloghook"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func (fw *Framework) openLogfile() error {
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return err
	}

	// if its the server and we are not interactive we log to event log
	if fw.Config.InitiatedByServer && !interactive {
		elog, err := eventlog.Open("choria-server")
		if err != nil {
			return err
		}

		fw.log.AddHook(eventloghook.NewHook(elog))
	}

	// if its not the server we log to whatever is configured
	return fw.commonLogOpener()
}
