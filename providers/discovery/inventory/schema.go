// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"encoding/json"

	iu "github.com/choria-io/go-choria/internal/util"
)

func ValidateInventory(i []byte) (warnings []string, err error) {
	var d any
	err = json.Unmarshal(i, &d)
	if err != nil {
		return nil, err
	}

	return iu.ValidateSchemaFromFS("schemas/choria/discovery/v1/inventory_file.json", d)
}
