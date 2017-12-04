package data

// RegistrationItem contains a single registration message
type RegistrationItem struct {
	// Data is the raw data to publish
	Data *[]byte

	// Destination is unused but will let you set custom NATS targets
	Destination string

	// TargetAgent lets you pick where to send the data as a request
	TargetAgent string
}
