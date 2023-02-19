// Copyright (c) 2020-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package nagioswatcher

import (
	"fmt"
	"math"
	"time"

	"github.com/choria-io/go-choria/statistics"
)

func (w *Watcher) watchUsingChoria() (state State, output string, err error) {
	f, freq := w.machine.ChoriaStatusFile()
	if f == "" || freq == 0 {
		return UNKNOWN, "Status file not configured", nil
	}

	status, err := statistics.LoadInstanceStatus(f)
	if err != nil {
		return CRITICAL, fmt.Sprintf("Status file error: %s", err), nil
	}

	ce := math.MaxInt
	te := math.MaxInt

	if !status.CertificateExpires.IsZero() {
		ce = int(time.Until(status.CertificateExpires).Seconds())
	}
	if !status.TokenExpires.IsZero() {
		te = int(time.Until(status.TokenExpires).Seconds())
	}

	perfData := fmt.Sprintf("uptime=%d;; filtered_msgs=%d;; invalid_msgs=%d;; passed_msgs=%d;; replies_msgs=%d;; total_msgs=%d;; ttlexpired_msgs=%d;; last_msg=%d;; cert_expire_seconds=%d;; token_expire_seconds=%d;; events=%d;;", status.Uptime, int(status.Stats.Filtered), int(status.Stats.Invalid), int(status.Stats.Passed), int(status.Stats.Replies), int(status.Stats.Total), int(status.Stats.TTLExpired), status.LastMessage, ce, te, int(status.Stats.Events))

	err = status.CheckFileAge(time.Duration(3*freq) * time.Second)
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", err, perfData), nil
	}

	if w.properties.CertExpiry > 0 {
		err = status.CheckCertValidity(w.properties.CertExpiry)
		if err != nil {
			return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", err, perfData), nil
		}
	}

	if w.properties.TokenExpiry > 0 {
		err = status.CheckTokenValidity(w.properties.TokenExpiry)
		if err != nil {
			return CRITICAL, fmt.Sprintf("CRITICAL: %s|%s", err, perfData), nil
		}
	}

	err = status.CheckLastMessage(w.properties.LastMessage)
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: %s|%v", err, perfData), nil
	}

	err = status.CheckConnection()
	if err != nil {
		return CRITICAL, fmt.Sprintf("CRITICAL: Not connected to any server|%v", perfData), nil
	}

	return OK, fmt.Sprintf("OK: %s|%v", f, perfData), nil
}
