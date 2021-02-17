package stream

//
// import (
// 	"testing"
// 	"time"
//
// 	"github.com/golang/mock/gomock"
// 	"github.com/nats-io/jsm.go"
// 	"github.com/nats-io/nats-server/v2/server"
// 	"github.com/nats-io/nats.go"
// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// 	"github.com/sirupsen/logrus"
// )
//
// func Test(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "Registration")
// }
//
// type thing struct {
// 	H       string `json:"hello"`
// 	Updated chan struct{}
// }
//
// func (t *thing) Instance() interface{} {
// 	return &thing{}
// }
//
// func (t *thing) Update(u interface{}) {
// 	t.H = u.(*thing).H
// 	t.Updated <- struct{}{}
// }
//
// var _ = Describe("UpdateManager", func() {
// 	var (
// 		log     *logrus.Entry
// 		mockctl *gomock.Controller
// 		fw      *MockFramework
// 	)
//
// 	BeforeEach(func() {
// 		logger := logrus.New()
// 		// logger.SetOutput(GinkgoWriter)
// 		log = logrus.NewEntry(logger)
// 		mockctl = gomock.NewController(GinkgoT())
// 		fw = NewMockFramework(mockctl)
//
// 		fw.EXPECT().Logger(gomock.Any()).Return(log)
// 	})
//
// 	AfterEach(func() {
// 		mockctl.Finish()
// 	})
//
// 	It("Should update with the correct type", func() {
// 		srv, err := server.NewServer(&server.Options{JetStream: true, Port: -1})
// 		Expect(err).ToNot(HaveOccurred())
// 		go srv.Start()
// 		if !srv.ReadyForConnections(5 * time.Second) {
// 			Fail("jetstream did not become ready")
// 		}
//
// 		toSleep := func(n chan struct{}) {
// 			timer := time.NewTimer(time.Second)
// 			defer timer.Stop()
//
// 			select {
// 			case <-timer.C:
// 				Fail("timeout waiting for update")
// 			case <-n:
// 			}
// 		}
//
// 		nc, err := nats.Connect(srv.ClientURL())
// 		Expect(err).ToNot(HaveOccurred())
//
// 		jmgr, err := jsm.New(nc)
// 		Expect(err).ToNot(HaveOccurred())
//
// 		fw.EXPECT().NATSConn().Return(nc).AnyTimes()
//
// 		_, err = jmgr.NewStream("TEST", jsm.MemoryStorage(), jsm.Subjects("js.test.in"))
// 		Expect(err).ToNot(HaveOccurred())
//
// 		_, err = nc.Request("js.test.in", []byte(`{"hello":"world"}`), time.Second)
// 		Expect(err).ToNot(HaveOccurred())
//
// 		id := thing{Updated: make(chan struct{}, 1)}
//
// 		mgr, err := New("TEST", "", fw)
// 		Expect(err).ToNot(HaveOccurred())
// 		defer mgr.Close()
//
// 		err = mgr.Manage(&id)
// 		Expect(err).ToNot(HaveOccurred())
//
// 		toSleep(id.Updated)
//
// 		Expect(id.H).To(Equal("world"))
//
// 		_, err = nc.Request("js.test.in", []byte(`{"hello":"bob"}`), time.Second)
// 		Expect(err).ToNot(HaveOccurred())
//
// 		toSleep(id.Updated)
//
// 		Expect(id.H).To(Equal("bob"))
// 	})
// })
