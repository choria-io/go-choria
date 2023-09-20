// Copyright (c) 2020-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package audit

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	. "github.com/onsi/gomega"
)

func checkFileGid(stat os.FileInfo, group string) {
	gid := stat.Sys().(*syscall.Stat_t).Gid
	grp, err := user.LookupGroup(group)
	Expect(err).ToNot(HaveOccurred())

	expectedGid, err := strconv.Atoi(grp.Gid)
	Expect(err).ToNot(HaveOccurred())

	Expect(int(gid)).To(Equal(expectedGid))
}
