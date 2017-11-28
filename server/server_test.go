package server

import (
	"errors"
	"testing"

	"github.com/choria-io/go-choria/choria"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type StubPublishingConnector struct {
	PublishedMsgs []*choria.Message

	nextErr error
}

func (st *StubPublishingConnector) Publish(msg *choria.Message) error {
	st.PublishedMsgs = append(st.PublishedMsgs, msg)

	var err error

	if st.nextErr != nil {
		err = st.nextErr
		st.nextErr = nil
	}

	return err
}

func (st *StubPublishingConnector) SetNextError(err string) {
	st.nextErr = errors.New(err)
}

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server")
}
