// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"

	"github.com/choria-io/go-choria/providers/data"
	"github.com/choria-io/go-choria/providers/data/ddl"
)

func (srv *Instance) StartDataProviders(ctx context.Context) (err error) {
	if srv.fw.ProvisionMode() {
		return
	}

	srv.data, err = data.NewManager(ctx, srv.fw)
	if err != nil {
		return err
	}

	return nil
}

// DataFuncMap returns the list of known data plugins
func (srv *Instance) DataFuncMap() (ddl.FuncMap, error) {
	if srv.fw.ProvisionMode() {
		return nil, fmt.Errorf("data providers not available in provisioning mode")
	}

	return srv.data.FuncMap(srv)
}
