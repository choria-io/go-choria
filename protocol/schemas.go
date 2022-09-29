// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protocol

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/choria-io/go-choria/internal/fs"
	iu "github.com/choria-io/go-choria/internal/util"
)

var (
	// ErrSchemaUnknown indicates the schema could not be found
	ErrSchemaUnknown = errors.New("unknown schema")
	// ErrSchemaValidationFailed indicates that the validator failed to perform validation, perhaps due to invalid schema
	ErrSchemaValidationFailed = errors.New("validation failed")
)

// SchemaBytes returns the JSON schema matching a specific protocol definition like `ReplyV1`
func SchemaBytes(protocol string) ([]byte, error) {
	path, err := schemaPath(protocol)
	if err != nil {
		return nil, err
	}

	return fs.FS.ReadFile(path)
}

func schemaPath(protocol string) (string, error) {
	switch protocol {
	case ReplyV1:
		return "schemas/choria/protocol/v1/reply.json", nil
	case RequestV1:
		return "schemas/choria/protocol/v1/request.json", nil
	case SecureReplyV1:
		return "schemas/choria/protocol/v1/secure_reply.json", nil
	case SecureRequestV1:
		return "schemas/choria/protocol/v1/secure_request.json", nil
	case TransportV1:
		return "schemas/choria/protocol/v1/transport.json", nil
	default:
		return "", ErrSchemaUnknown
	}
}

// SchemaValidate validates data against the JSON schema for protocol
func SchemaValidate(protocol string, data []byte) (valid bool, errors []string, err error) {
	paht, err := schemaPath(protocol)
	if err != nil {
		return false, nil, err
	}

	var d any
	err = json.Unmarshal(data, &d)
	if err != nil {
		return false, nil, fmt.Errorf("%w: invalid json data: %v", ErrSchemaValidationFailed, err)
	}

	errors, err = iu.ValidateSchemaFromFS(paht, d)
	switch err {
	case nil:
		return len(errors) == 0, errors, nil
	case iu.ErrSchemaValidationFailed:
		return false, nil, ErrSchemaValidationFailed
	case iu.ErrSchemaUnknown:
		return false, nil, ErrSchemaUnknown
	}

	return len(errors) == 0, errors, nil
}
