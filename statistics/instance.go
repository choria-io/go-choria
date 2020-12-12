package statistics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// ServerStats are internal statistics about the running server
type ServerStats struct {
	Total      float64 `json:"total"`
	Valid      float64 `json:"valid"`
	Invalid    float64 `json:"invalid"`
	Passed     float64 `json:"passed"`
	Filtered   float64 `json:"filtered"`
	Replies    float64 `json:"replies"`
	TTLExpired float64 `json:"ttlexpired"`
}

// InstanceStatus describes the current instance status
type InstanceStatus struct {
	Identity        string       `json:"identity"`
	Uptime          int64        `json:"uptime"`
	ConnectedServer string       `json:"connected_server"`
	LastMessage     int64        `json:"last_message"`
	Provisioning    bool         `json:"provisioning_mode"`
	Stats           *ServerStats `json:"stats"`
	FileName        string       `json:"-"`
	ModTime         time.Time    `json:"-"`
}

func LoadInstanceStatus(f string) (*InstanceStatus, error) {
	raw, err := ioutil.ReadFile(f)
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

func (i *InstanceStatus) CheckFileAge(maxAge time.Duration) error {
	if i.ModTime.Before(time.Now().Add(-1 * maxAge)) {
		return fmt.Errorf("older than %v", maxAge)
	}

	return nil
}

func (i *InstanceStatus) CheckLastMessage(maxAge time.Duration) error {
	previous := time.Unix(i.LastMessage, 0)
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
