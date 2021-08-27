package inter

// ConnectorMessage is received from middleware
type ConnectorMessage interface {
	Subject() string
	Reply() string
	Data() []byte

	// Msg is the middleware specific message like *nats.Msg
	Msg() interface{}
}
