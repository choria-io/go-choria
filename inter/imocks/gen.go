// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

package main

//go:generate mockgen -destination connector.go -package imock -source ../connector.go
//go:generate mockgen -destination connector_message.go -package imock -source ../connector_message.go
//go:generate mockgen -destination connector_info.go -package imock -source ../connector_info.go
//go:generate mockgen -destination request_signer.go -package imock -source ../request_signer.go
//go:generate mockgen -destination framework.go -package imock github.com/choria-io/go-choria/inter ConfigurationProvider,ProtocolConstructor,ConnectionManager,Framework
//go:generate mockgen -destination ddl_resolver.go -package imock github.com/choria-io/go-choria/inter DDLResolver
//go:generate mockgen -destination security.go -package imock github.com/choria-io/go-choria/inter SecurityProvider

func main() {}
