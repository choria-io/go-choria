// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"context"
)

// DDLResolver allows DDLs to be found on the local system or via remote registries
type DDLResolver interface {
	// String indicates which resolver is in use
	String() string

	// DDL resolves a DDL and unmarshal it into target, which should be a pointer to a DDL type
	DDL(ctx context.Context, kind string, name string, target interface{}, fw Framework) error

	// DDLBytes resolves a DDL and return the bytes
	DDLBytes(ctx context.Context, kind string, name string, fw Framework) ([]byte, error)

	// DDLNames returns all the DDL names this resolver knows about
	DDLNames(ctx context.Context, kind string, w Framework) ([]string, error)
}
