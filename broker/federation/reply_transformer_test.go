package federation

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/mcollective"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReplyTransformer", func() {
	var (
		choria      *mcollective.Choria
		request     protocol.Request
		reply       protocol.Reply
		sreply      protocol.SecureReply
		transformer ReplyTransformer
		in          chainmessage
		err         error
	)

	BeforeEach(func() {
		log.SetOutput(ioutil.Discard)

		choria, err = mcollective.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		request, err = choria.NewRequest(protocol.RequestV1, "test", "tester", "choria=tester", 60, choria.NewRequestID(), "mcollective")
		Expect(err).ToNot(HaveOccurred())
		request.SetMessage(`{"hello":"world"}`)

		reply, err = choria.NewReply(request)
		Expect(err).ToNot(HaveOccurred())

		sreply, err = choria.NewSecureReply(reply)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = choria.NewTransportForSecureReply(sreply)
		Expect(err).ToNot(HaveOccurred())

		err = transformer.Init("testing", "1")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should correctly transform a message", func() {
		tr, err := choria.NewTransportForSecureReply(sreply)
		Expect(err).ToNot(HaveOccurred())

		tr.SetFederationRequestID(request.RequestID())
		tr.SetFederationReplyTo("mcollective.reply")

		in.Message = tr
		in.RequestID = reply.RequestID()

		out, err := transformer.process(in, "tester:1")
		Expect(err).ToNot(HaveOccurred())

		Expect(out.Targets).To(Equal([]string{"mcollective.reply"}))

		id, federated := out.Message.FederationRequestID()
		Expect(id).To(BeEmpty())
		Expect(federated).To(BeFalse())
	})

	It("should fail for unfederated messages", func() {
		_, err = transformer.process(in, "tester:1")

		Expect(err).To(MatchError("tester:1 received a message from rip.mcollective that is not federated"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")

		_, err = transformer.process(in, "tester:1")

		Expect(err).To(MatchError("tester:1 received a message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})
})
