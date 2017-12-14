package build

// Version the application version
var Version = "development"

// SHA is the git reference used to build this package
var SHA = "unknown"

// BuildDate is when it was build
var BuildDate = "unknown"

// License is the official Open Source Initiave license abbreciation
var License = "Apache-2.0"

// TLS controls the NATS protocol level TLS
var TLS = "true"

// Secure controls the signing and validations of certificates in the protocol
var Secure = "true"

// IsSecure determines if this build will validate senders at protocol level
func IsSecure() bool {
	return Secure == "true"
}

// HasTLS determines if TLS should be used on the wire
func HasTLS() bool {
	return TLS == "true"
}
