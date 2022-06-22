mockgen -destination connector.go -package imock -source ../connector.go
mockgen -destination connector_message.go -package imock -source ../connector_message.go
mockgen -destination connector_info.go -package imock -source ../connector_info.go
mockgen -destination request_signer.go -package imock -source ../request_signer.go
mockgen -destination framework.go -package imock github.com/choria-io/go-choria/inter ConfigurationProvider,ProtocolConstructor,ConnectionManager,Framework
mockgen -destination ddl_resolver.go -package imock github.com/choria-io/go-choria/inter DDLResolver
mockgen -destination security.go -package imock github.com/choria-io/go-choria/inter SecurityProvider
