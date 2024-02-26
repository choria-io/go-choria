// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type FileReportHandler func(file string, ok bool)

var (
	sumsMatcherRe        = regexp.MustCompile(`^([a-z0-9]+)\s+(\S.+)$`)
	ErrorInvalidSumsFile = fmt.Errorf("invalid sums file")
	ErrorChecksumError   = fmt.Errorf("could not checksum file")
)

// Sha256HashFile computes the sha256 sum of a hash and returns the hex encoded result
func Sha256HashFile(path string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Sha256HashBytes computes the sha256 sum of the byes c and returns the hex encoded result
func Sha256HashBytes(c []byte) (string, error) {
	hasher := sha256.New()
	r := bytes.NewReader(c)
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// FileHasSha256Sum verifies that path has the checksum provided, sum should be hex encoded
func FileHasSha256Sum(path string, sum string) (bool, string, error) {
	s, err := Sha256HashFile(path)
	if err != nil {
		return false, "", err
	}

	return s == sum, s, nil
}

// Sha256ChecksumDir produce a file similar to those produced by sha256sum on the command line.
//
// This function will walk the directory and checksum all files in all subdirectories
func Sha256ChecksumDir(dir string) ([]byte, error) {
	sums := bytes.NewBuffer([]byte{})

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		sum, err := Sha256HashFile(path)
		if err != nil {
			return err
		}

		if dir != "." {
			path = strings.TrimPrefix(path, filepath.ToSlash(dir))
		}
		path = strings.TrimPrefix(path, "/")

		_, err = fmt.Fprintf(sums, "%s  %s\n", sum, path)

		return err
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrorChecksumError, err)
	}

	return sums.Bytes(), nil
}

// Sha256VerifyDir verifies the sums in sumsFile relative to dir, sumsFile should have been made using Sha256ChecksumDir
//
// When dir is supplied the files in that dir will be verified else the path will be relative to the path of the sumsFile
func Sha256VerifyDir(sumsFile string, dir string, log *logrus.Entry, cb FileReportHandler) (bool, error) {
	if dir == "" {
		abs, err := filepath.Abs(sumsFile)
		if err != nil {
			return false, fmt.Errorf("%w: could not determine directory for files: %v", ErrorChecksumError, err)
		}
		dir = filepath.Dir(abs)
	}

	sums, err := os.Open(sumsFile)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrorChecksumError, err)
	}
	defer sums.Close()

	scanner := bufio.NewScanner(sums)
	scanner.Split(bufio.ScanLines)

	lc := 0
	failed := false
	for scanner.Scan() {
		lc++

		line := scanner.Text()
		if log != nil {
			log.Debugf("Checking line: %v", line)
		}

		if len(line) == 0 {
			continue
		}

		matches := sumsMatcherRe.FindStringSubmatch(line)
		if len(matches) != 3 {
			return false, fmt.Errorf("%w: malformed line %d", ErrorInvalidSumsFile, lc)
		}

		if log != nil {
			log.Debugf("Checking file %v", matches[2])
		}

		if matches[2] == sumsFile {
			continue
		}

		ok, _, err := FileHasSha256Sum(filepath.Join(dir, matches[2]), matches[1])
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// let it fail validation without error
		case err != nil:
			return false, fmt.Errorf("%w: %v", ErrorChecksumError, err)
		}

		if log != nil {
			log.Debugf("File %v: %t", matches[2], ok)
		}

		if !ok {
			failed = true
		}

		if cb != nil {
			cb(matches[2], ok)
		}
	}

	return !failed, scanner.Err()
}
