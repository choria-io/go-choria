// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0
//go:build !windows
// +build !windows

package audit

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/onsi/gomega"
)

func checkFileGid(stat os.FileInfo, group string) {
	gid := stat.Sys().(*syscall.Stat_t).Gid
	grp, err := user.LookupGroup(group)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	expectedGid, err := strconv.Atoi(grp.Gid)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Expect(int(gid)).To(gomega.Equal(int(expectedGid)))
}
