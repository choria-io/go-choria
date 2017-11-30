package connectortest

import (
	"errors"

	"github.com/choria-io/go-choria/choria"
)

type PublishableConnector struct {
	PublishedMsgs []*choria.Message

	nextErr error
}

func (st *PublishableConnector) Publish(msg *choria.Message) error {
	st.PublishedMsgs = append(st.PublishedMsgs, msg)

	var err error

	if st.nextErr != nil {
		err = st.nextErr
		st.nextErr = nil
	}

	return err
}

func (st *PublishableConnector) SetNextError(err string) {
	st.nextErr = errors.New(err)
}
