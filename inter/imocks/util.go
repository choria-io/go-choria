package imock

import (
	"io"

	"github.com/brutella/hc/util"
	"github.com/choria-io/go-choria/config"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func NewFrameworkForTests(ctrl *gomock.Controller, logWriter io.Writer) (*MockFramework, *config.Config) {
	logger := logrus.New()
	logger.SetOutput(logWriter)

	fw := NewMockFramework(ctrl)
	fw.EXPECT().Configuration().Return(config.NewConfigForTests()).AnyTimes()
	fw.EXPECT().Logger(gomock.AssignableToTypeOf("")).Return(logrus.NewEntry(logger)).AnyTimes()
	// fw.EXPECT().ProvisionMode().Return(false).AnyTimes()
	fw.EXPECT().NewRequestID().Return(util.RandomHexString(), nil).AnyTimes()

	return fw, fw.Configuration()
}
