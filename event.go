package lifecycle

// Event is event that can be published to the network
type Event interface {
	Target() (string, error)
	String() string
	Type() Type
	TypeString() string
	SetIdentity(string)
	Component() string
	Identity() string
	ID() string
}
