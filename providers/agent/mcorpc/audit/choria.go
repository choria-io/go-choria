// Copyright (c) 2020-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package audit is a auditing system that's compatible with the
// one found in the mcollective-choria Ruby project, log lines will
// be identical and can be put in the same file as the ruby one
package audit

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
)

var mu = &sync.Mutex{}

// Message is the format of a Choria audit log
type Message struct {
	TimeStamp   string          `json:"timestamp"`
	RequestID   string          `json:"request_id"`
	RequestTime int64           `json:"request_time"`
	CallerID    string          `json:"caller"`
	Sender      string          `json:"sender"`
	Agent       string          `json:"agent"`
	Action      string          `json:"action"`
	Data        json.RawMessage `json:"data"`
}

// Request writes a audit log to a configured log
func Request(request protocol.Request, agent string, action string, data json.RawMessage, cfg *config.Config) bool {
	if !cfg.RPCAudit {
		return false
	}

	logfile := cfg.Choria.RPCAuditLogfile
	logfileGroup := cfg.Choria.RPCAuditLogfileGroup

	logfileMode, err := strconv.ParseUint(cfg.Choria.RPCAuditLogFileMode, 0, 32)
	if err != nil {
		log.Errorf("Failed to parse plugin.rpcaudit.logfile.mode: %v", err)
		return false
	}

	if logfile == "" {
		log.Warnf("Choria RPC Auditing is enabled but no logfile is configured, skipping")
		return false
	}

	amsg := Message{
		TimeStamp:   time.Now().UTC().Format("2006-01-02T15:04:05.000000-0700"),
		RequestID:   request.RequestID(),
		RequestTime: request.Time().UTC().Unix(),
		CallerID:    request.CallerID(),
		Sender:      request.SenderID(),
		Agent:       agent,
		Action:      action,
		Data:        data,
	}

	j, err := json.Marshal(amsg)
	if err != nil {
		log.Warnf("Auditing is not functional because the auditing data could not be represented as JSON: %s", err)
		return false
	}

	mu.Lock()
	defer mu.Unlock()

	f, err := createAuditLog(logfile, logfileGroup, uint32(logfileMode))
	if err != nil {
		log.Warnf("Auditing is not functional because opening the logfile '%s' failed: %s", logfile, err)
		return false
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s\n", string(j)))
	if err != nil {
		log.Warnf("Auditing is not functional because writing to logfile '%s' failed: %s", logfile, err)
		return false
	}

	return true
}

func createAuditLog(logfile string, logfileGroup string, logfileMode uint32) (*os.File, error) {
	f, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fs.FileMode(logfileMode))
	if err != nil {
		return f, err
	}

	if logfileGroup == "" {
		return f, nil
	}

	grp, err := user.LookupGroup(logfileGroup)
	if err != nil {
		f.Close()
		return f, err
	}

	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		f.Close()
		return f, err
	}

	err = os.Chown(logfile, os.Getuid(), gid)
	if err != nil {
		f.Close()
		return f, err
	}

	return f, nil
}
