package lifecycle

// Event is event that can be published to the network
type Event interface {
	Target() (string, error)
	String() string
	Component() string
	Type() Type
}
