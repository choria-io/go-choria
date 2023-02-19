// Copyright (c) 2020-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package statistics

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ServerStats are internal statistics about the running server
type ServerStats struct {
	Total      int64 `json:"total"`
	Valid      int64 `json:"valid"`
	Invalid    int64 `json:"invalid"`
	Passed     int64 `json:"passed"`
	Filtered   int64 `json:"filtered"`
	Replies    int64 `json:"replies"`
	TTLExpired int64 `json:"ttlexpired"`
	Events     int64 `json:"events"`
}

// InstanceStatus describes the current instance status
type InstanceStatus struct {
	Identity           string       `json:"identity"`
	Uptime             int64        `json:"uptime"`
	ConnectedServer    string       `json:"connected_server"`
	LastMessage        int64        `json:"last_message"`
	Provisioning       bool         `json:"provisioning_mode"`
	Stats              *ServerStats `json:"stats"`
	CertificateExpires time.Time    `json:"certificate_expires"`
	TokenExpires       time.Time    `json:"token_expires"`
	FileName           string       `json:"-"`
	ModTime            time.Time    `json:"-"`
}

func LoadInstanceStatus(f string) (*InstanceStatus, error) {
	raw, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}

	status := &InstanceStatus{}
	err = json.Unmarshal(raw, status)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(f)
	if err != nil {
		return nil, err
	}

	status.FileName = f
	status.ModTime = stat.ModTime()

	return status, nil
}

func (i *InstanceStatus) CheckTokenValidity(tillExpire time.Duration) error {
	if i.TokenExpires.IsZero() {
		return nil
	}

	if time.Until(i.TokenExpires) < tillExpire {
		return fmt.Errorf("token expires %v (%v)", i.TokenExpires, time.Until(i.TokenExpires).Round(time.Second))
	}

	return nil
}

func (i *InstanceStatus) CheckCertValidity(tillExpire time.Duration) error {
	if i.CertificateExpires.IsZero() {
		return nil
	}

	if time.Until(i.CertificateExpires) < tillExpire {
		return fmt.Errorf("certificate expires %v (%v)", i.CertificateExpires, time.Until(i.CertificateExpires).Round(time.Second))
	}

	return nil
}

func (i *InstanceStatus) CheckFileAge(maxAge time.Duration) error {
	if i.ModTime.Before(time.Now().Add(-1 * maxAge)) {
		return fmt.Errorf("older than %v", maxAge)
	}

	return nil
}

func (i *InstanceStatus) CheckLastMessage(maxAge time.Duration) error {
	previous := time.Unix(i.LastMessage, 0)

	// we don't check the first maxAge if uptime is low and we never had any messages
	// to avoid large sets of critical after restarts, upgrades etc
	if i.LastMessage == 0 {
		if float64(i.Uptime) < maxAge.Seconds() {
			return nil
		}
	}

	if previous.Before(time.Now().Add(-1 * maxAge)) {
		return fmt.Errorf("last message at %v", previous.UTC())
	}

	return nil
}

// CheckConnection checks if the server is currently connected
func (i *InstanceStatus) CheckConnection() error {
	if i.ConnectedServer == "" {
		return fmt.Errorf("not connected")
	}

	return nil
}
