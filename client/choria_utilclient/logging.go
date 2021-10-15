// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

// generated code; DO NOT EDIT

package choria_utilclient

func (c *ChoriaUtilClient) infof(msg string, a ...interface{}) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaUtilClient) errorf(msg string, a ...interface{}) {
	c.clientOpts.logger.Errorf(msg, a...)
}
