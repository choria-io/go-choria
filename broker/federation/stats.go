package federation

import "time"

type Stats struct {
	Version         *string      `json:"version"` // TODO
	StartTime       time.Time    `json:"start_time"`
	ClusterName     *string      `json:"cluster"`
	ClusterInstance *string      `json:"instance"`
	ConfigFile      *string      `json:"config_file"`
	Status          string       `json:"status"`
	CollectiveStats *WorkerStats `json:"collective"`
	FederationStats *WorkerStats `json:"federation"`
}

type WorkerStats struct {
	Source          *string   `json:"source"`
	Received        int       `json:"received"`
	Sent            int       `json:"sent"`
	LastMessage     time.Time `json:"last_message"`
	ConnectedServer string    `json:"connected_server"`
}
