// Copyright (c) 2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package execwatcher

import (
	"fmt"
	"os/exec"
)

func disown(cmd *exec.Cmd) error {
	return fmt.Errorf("disown is not supported on windows")
}
