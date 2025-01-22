// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

func abort(reply *mcorpc.Reply, format string, a ...any) {
	reply.Statuscode = mcorpc.Aborted
	reply.Statusmsg = fmt.Sprintf(format, a...)
}
