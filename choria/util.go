// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"fmt"
	"strings"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// UserConfig determines what is the active config file for a user
func UserConfig() string {
	return util.UserConfig()
}

// BuildInfo retrieves build information
func BuildInfo() *build.Info {
	return util.BuildInfo()
}

// FileExist checks if a file exist
func FileExist(path string) bool {
	return util.FileExist(path)
}

// NewRequestID Creates a new v1 RequestID like random string. Here for backwards compat with older clients
func NewRequestID() (string, error) {
	return strings.Replace(util.UniqueID(), "-", "", -1), nil
}

// ParseDuration is an extended version of go duration parsing that
// also supports w,W,d,D,M,Y,y in addition to what go supports
func ParseDuration(dstr string) (dur time.Duration, err error) {
	return util.ParseDuration(dstr)
}

// FileIsRegular tests if a file is a regular file, no links, etc
func FileIsRegular(path string) bool {
	return util.FileIsRegular(path)
}

// FileIsDir tests if a file is a directory
func FileIsDir(path string) bool {
	return util.FileIsDir(path)
}

// NatsConnectionHelpers constructs token based private inbox and helpers for the nats.UserJWT() function. Only Server and Client tokens are supported.
func NatsConnectionHelpers(token string, collective string, seedFile string, log *logrus.Entry) (inbox string, jwth nats.UserJWTHandler, sigh nats.SignatureHandler, err error) {
	if collective == "" {
		return "", nil, nil, fmt.Errorf("collective is required")
	}

	if seedFile == "" {
		return "", nil, nil, fmt.Errorf("seedfile is required")
	}

	purpose := tokens.TokenPurpose(token)

	var uid string
	var isExp func() bool
	var exp time.Time

	switch purpose {
	case tokens.ClientIDPurpose:
		client, err := tokens.ParseClientIDTokenUnverified(token)
		if err != nil {
			return "", nil, nil, err
		}
		_, uid = client.UniqueID()
		isExp = client.IsExpired
		exp = client.ExpireTime()

	case tokens.ServerPurpose:
		server, err := tokens.ParseServerTokenUnverified(token)
		if err != nil {
			return "", nil, nil, err
		}
		_, uid = server.UniqueID()
		isExp = server.IsExpired
		exp = server.ExpireTime()

	default:
		return "", nil, nil, fmt.Errorf("unsupported token purpose: %v", purpose)
	}

	inbox = fmt.Sprintf("%s.reply.%s", collective, uid)

	jwth = func() (string, error) {
		if isExp() {
			log.Errorf("Cannot sign connection NONCE: token is expired by %v", time.Since(exp))
			return "", fmt.Errorf("token expired")
		}
		return token, nil
	}

	sigh = func(n []byte) ([]byte, error) {
		if isExp() {
			log.Errorf("Cannot sign connection NONCE: token is expired by %v", time.Since(exp))
			return nil, fmt.Errorf("token expired")
		}
		log.Debugf("Signing nonce using seed file %s", seedFile)
		return Ed25519SignWithSeedFile(seedFile, n)
	}

	return inbox, jwth, sigh, nil
}
