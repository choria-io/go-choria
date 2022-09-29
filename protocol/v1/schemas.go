// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"

	"github.com/choria-io/go-choria/protocol"
)

func schemaValidate(version string, data []byte) (valid bool, errs []string, err error) {
	valid, errs, err = protocol.SchemaValidate(version, data)

	switch {
	case errors.Is(err, protocol.ErrSchemaValidationFailed):
		badJsonCtr.Inc()
		protocolErrorCtr.Inc()
	case !valid:
		protocolErrorCtr.Inc()
		invalidCtr.Inc()
	}

	return valid, errs, err
}
