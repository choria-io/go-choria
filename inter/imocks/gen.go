// Copyright (c) 2023-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

package main

//go:generate mockgen -write_generate_directive -destination connector.go -package imock -source ../connector.go
//go:generate mockgen -write_generate_directive -destination connector_message.go -package imock -source ../connector_message.go
//go:generate mockgen -write_generate_directive -destination connector_info.go -package imock -source ../connector_info.go
//go:generate mockgen -write_generate_directive -destination request_signer.go -package imock -source ../request_signer.go
//go:generate mockgen -write_generate_directive -destination framework.go -package imock github.com/choria-io/go-choria/inter ConfigurationProvider,ProtocolConstructor,ConnectionManager,Framework
//go:generate mockgen -write_generate_directive -destination ddl_resolver.go -package imock github.com/choria-io/go-choria/inter DDLResolver
//go:generate mockgen -write_generate_directive -destination security.go -package imock github.com/choria-io/go-choria/inter SecurityProvider

func main() {}
