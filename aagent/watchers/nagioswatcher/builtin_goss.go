package nagioswatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aelsabbahy/goss"
	"github.com/aelsabbahy/goss/outputs"
	gossutil "github.com/aelsabbahy/goss/util"
)

func (w *Watcher) watchUsingGoss(vars string) (state State, output string, err error) {
	var out bytes.Buffer

	opts := []gossutil.ConfigOption{
		gossutil.WithMaxConcurrency(1),
		gossutil.WithResultWriter(&out),
		gossutil.WithSpecFile(w.gossFile),
	}

	if vars != "" {
		_, err := os.Stat(vars)
		if err == nil {
			opts = append(opts, gossutil.WithVarsFile(vars))
		}
	}

	cfg, err := gossutil.NewConfig(opts...)
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss configuration failed: %s", err), err
	}

	_, err = goss.Validate(cfg, time.Now())
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss validate failed: %s", err), err
	}

	res := &outputs.StructuredOutput{}
	err = json.Unmarshal(out.Bytes(), res)
	if err != nil {
		return UNKNOWN, fmt.Sprintf("UNKNOWN: goss output invalid: %s", err), err
	}

	pd := fmt.Sprintf("checks=%d;; failed=%d;; runtime=%fs", res.Summary.TestCount, res.Summary.Failed, res.Summary.TotalDuration.Seconds())

	if res.Summary.Failed > 0 {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", res.SummaryLine, pd), nil
	}

	return OK, fmt.Sprintf("OK: %s|%s", res.SummaryLine, pd), nil
}
