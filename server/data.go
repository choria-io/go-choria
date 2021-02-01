package server

import (
	"context"

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
	return srv.data.FuncMap(srv)
}
