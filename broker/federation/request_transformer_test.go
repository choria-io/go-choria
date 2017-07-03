package federation

import (
	"io/ioutil"

	"github.com/choria-io/go-choria/mcollective"
	"github.com/choria-io/go-choria/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("RequestTransformer", func() {
	var (
		choria      *mcollective.Choria
		request     protocol.Request
		srequest    protocol.SecureRequest
		transformer RequestTransformer
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

		srequest, err = choria.NewSecureRequest(request)
		Expect(err).ToNot(HaveOccurred())

		in.Message, err = choria.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		err = transformer.Init("testing", "1")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should correctly transform a message", func() {
		tr, err := choria.NewTransportForSecureRequest(srequest)
		Expect(err).ToNot(HaveOccurred())

		tr.SetFederationRequestID(request.RequestID())
		tr.SetFederationTargets([]string{"mcollective.discovery"})
		tr.SetReplyTo("mcollective.reply")

		in.Message = tr
		in.RequestID = request.RequestID()

		out, err := transformer.process(in, "tester:1")
		Expect(err).ToNot(HaveOccurred())

		Expect(out.Message.ReplyTo()).To(Equal("federation.reply.target"))

		id, _ := out.Message.FederationRequestID()
		Expect(id).To(Equal(request.RequestID()))

		replyto, _ := out.Message.FederationReplyTo()
		Expect("mcollective.reply").To(Equal(replyto))

		targets, _ := out.Message.FederationTargets()
		Expect(targets).To(BeEmpty())
		Expect(out.Targets).To(Equal([]string{"mcollective.discovery"}))
	})

	It("should fail for unfederated messages", func() {
		_, err = transformer.process(in, "tester:1")

		Expect(err).To(MatchError("tester:1 received a message from rip.mcollective that is not federated"))
	})

	It("Should fail for messages with no targets", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		_, err = transformer.process(in, "tester:1")

		Expect(err).To(MatchError("tester:1 received a message 80a1ac20463745c0b12cfe6e3db61dff from rip.mcollective that does not have any targets"))
	})

	It("Should fail for messages with no reply-to", func() {
		in.Message.SetFederationRequestID("80a1ac20463745c0b12cfe6e3db61dff")
		in.Message.SetFederationTargets([]string{"reply.1"})

		_, err = transformer.process(in, "tester:1")

		Expect(err).To(MatchError("tester:1 received a message 80a1ac20463745c0b12cfe6e3db61dff with no reply-to set"))
	})
})
