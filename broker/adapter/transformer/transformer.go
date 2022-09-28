// Copyright (c) 2019-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package transformer

import (
	"time"

	"github.com/choria-io/go-choria/broker/adapter/ingest"
)

type Msg struct {
	Protocol  string    `json:"protocol"`
	Data      string    `json:"data"`
	Sender    string    `json:"sender"`
	Time      time.Time `json:"time"`
	RequestID string    `json:"requestid"`
}

func TransformToOutput(msg ingest.Adaptable, adapterName string) *Msg {
	return &Msg{
		Protocol:  "choria:adapters:" + adapterName + ":output:1",
		Data:      string(msg.Message()),
		Sender:    msg.SenderID(),
		Time:      msg.Time().UTC(),
		RequestID: msg.RequestID(),
	}
}
