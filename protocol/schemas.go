// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protocol

import (
	"errors"
	"fmt"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/xeipuuv/gojsonschema"
)

var (
	// ErrSchemaUnknown indicates the schema could not be found
	ErrSchemaUnknown = errors.New("unknown schema")
	// ErrSchemaValidationFailed indicates that the validator failed to perform validation, perhaps due to invalid schema
	ErrSchemaValidationFailed = errors.New("validation failed")
)

// SchemaBytes returns the JSON schema matching a specific protocol definition like `ReplyV1`
func SchemaBytes(protocol string) ([]byte, error) {
	switch protocol {
	case ReplyV1:
		return fs.FS.ReadFile("protocol/v1/reply.json")
	case RequestV1:
		return fs.FS.ReadFile("protocol/v1/request.json")
	case SecureReplyV1:
		return fs.FS.ReadFile("protocol/v1/secure_reply.json")
	case SecureRequestV1:
		return fs.FS.ReadFile("protocol/v1/secure_request.json")
	case TransportV1:
		return fs.FS.ReadFile("protocol/v1/transport.json")
	default:
		return nil, ErrSchemaUnknown
	}
}

// SchemaValidate validates data against the JSON schema for protocol
func SchemaValidate(protocol string, data []byte) (valid bool, errors []string, err error) {
	schema, err := SchemaBytes(protocol)
	if err != nil {
		return false, nil, err
	}

	js := gojsonschema.NewBytesLoader(schema)
	d := gojsonschema.NewBytesLoader(data)

	validation, err := gojsonschema.Validate(js, d)
	if err != nil {
		return false, errors, fmt.Errorf("%w: %v", ErrSchemaValidationFailed, err)
	}

	if !validation.Valid() {
		for _, desc := range validation.Errors() {
			errors = append(errors, desc.String())
		}

		return false, errors, nil
	}

	return true, errors, nil
}
