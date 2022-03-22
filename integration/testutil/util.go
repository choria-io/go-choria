// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"fmt"
	"path/filepath"

	"github.com/onsi/gomega/gbytes"
	"github.com/sirupsen/logrus"
)

func GbytesLogger(level logrus.Level) (*gbytes.Buffer, *logrus.Logger) {
	logger := logrus.New()
	logger.SetLevel(level)
	buffer := gbytes.NewBuffer()
	logger.SetOutput(buffer)

	return buffer, logger
}

func CertPath(ca string, certname string) string {
	path, _ := filepath.Abs(filepath.Join("../../ca", ca, fmt.Sprintf("certs/%s.pem", certname)))
	return path
}

func KeyPath(ca string, certname string) string {
	path, _ := filepath.Abs(filepath.Join("../../ca", ca, fmt.Sprintf("%s-key.pem", certname)))
	return path
}
