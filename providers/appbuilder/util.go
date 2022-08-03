// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package appbuilder

import (
	"github.com/choria-io/appbuilder/builder"
	"github.com/choria-io/go-choria/client/discovery"
)

func ProcessStdDiscoveryOptions(f *discovery.StandardOptions, arguments map[string]any, flags map[string]any, config any) error {
	var err error

	if f.DiscoveryMethod != "" {
		f.DiscoveryMethod, err = builder.ParseStateTemplate(f.DiscoveryMethod, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for k, v := range f.DiscoveryOptions {
		f.DiscoveryOptions[k], err = builder.ParseStateTemplate(v, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	if f.Collective != "" {
		f.Collective, err = builder.ParseStateTemplate(f.Collective, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	if f.NodesFile != "" {
		f.NodesFile, err = builder.ParseStateTemplate(f.NodesFile, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	if f.CompoundFilter != "" {
		f.CompoundFilter, err = builder.ParseStateTemplate(f.CompoundFilter, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.CombinedFilter {
		f.CombinedFilter[i], err = builder.ParseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.IdentityFilter {
		f.IdentityFilter[i], err = builder.ParseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.AgentFilter {
		f.AgentFilter[i], err = builder.ParseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.ClassFilter {
		f.ClassFilter[i], err = builder.ParseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	for i, item := range f.FactFilter {
		f.FactFilter[i], err = builder.ParseStateTemplate(item, arguments, flags, config)
		if err != nil {
			return err
		}
	}

	return nil
}
