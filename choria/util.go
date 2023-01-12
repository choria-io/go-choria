// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"strings"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/internal/util"
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
