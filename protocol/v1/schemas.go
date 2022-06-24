// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	_ "embed"
	"fmt"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/xeipuuv/gojsonschema"
)

var (
	replySchema, _         = fs.FS.ReadFile("protocol/v1/reply.json")
	requestSchema, _       = fs.FS.ReadFile("protocol/v1/request.json")
	secureReplySchema, _   = fs.FS.ReadFile("protocol/v1/secure_reply.json")
	secureRequestSchema, _ = fs.FS.ReadFile("protocol/v1/secure_request.json")
	transportSchema, _     = fs.FS.ReadFile("protocol/v1/transport.json")
)

func schemaValidate(schema []byte, data string) (result bool, errors []string, err error) {
	if len(schema) == 0 {
		return false, nil, fmt.Errorf("invalid schema")
	}

	js := gojsonschema.NewStringLoader(string(schema))
	d := gojsonschema.NewStringLoader(data)

	validation, err := gojsonschema.Validate(js, d)
	if err != nil {
		badJsonCtr.Inc()
		protocolErrorCtr.Inc()

		return false, errors, fmt.Errorf("could not validate incoming document: %s", err)
	}

	if !validation.Valid() {
		protocolErrorCtr.Inc()
		invalidCtr.Inc()
		for _, desc := range validation.Errors() {
			errors = append(errors, desc.String())
		}

		return false, errors, nil
	}

	return true, errors, nil
}
