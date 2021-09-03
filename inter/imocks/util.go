package imock

import (
	"io"
	"strings"

	"github.com/brutella/hc/util"
	"github.com/choria-io/go-choria/config"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

type fwMockOpts struct {
	callerID   string
	logDiscard bool
	cfg        *config.Config
}
type fwMockOption func(*fwMockOpts)

func WithCallerID(c ...string) fwMockOption {
	return func(o *fwMockOpts) {
		if len(c) == 0 {
			o.callerID = "choria=rip.mcollective"
		} else {
			o.callerID = c[0]
		}
	}
}

func LogDiscard() fwMockOption {
	return func(o *fwMockOpts) {
		o.logDiscard = true
	}
}

func WithConfig(c *config.Config) fwMockOption {
	return func(o *fwMockOpts) { o.cfg = c }
}

func WithConfigFile(f string) fwMockOption {
	return func(o *fwMockOpts) {
		cfg, err := config.NewConfig(f)
		if err != nil {
			panic(err)
		}
		o.cfg = cfg
	}
}

func NewFrameworkForTests(ctrl *gomock.Controller, logWriter io.Writer, opts ...fwMockOption) (*MockFramework, *config.Config) {
	mopts := &fwMockOpts{cfg: config.NewConfigForTests()}
	for _, o := range opts {
		o(mopts)
	}

	logger := logrus.New()
	if mopts.logDiscard {
		logger.SetOutput(io.Discard)
	} else {
		logger.SetOutput(logWriter)
	}

	fw := NewMockFramework(ctrl)
	fw.EXPECT().Configuration().Return(mopts.cfg).AnyTimes()
	fw.EXPECT().Logger(gomock.AssignableToTypeOf("")).Return(logrus.NewEntry(logger)).AnyTimes()
	fw.EXPECT().NewRequestID().Return(util.RandomHexString(), nil).AnyTimes()
	fw.EXPECT().HasCollective(gomock.AssignableToTypeOf("")).DoAndReturn(func(c string) bool {
		for _, collective := range fw.Configuration().Collectives {
			if c == collective {
				return true
			}
		}
		return false
	}).AnyTimes()

	if mopts.callerID != "" {
		fw.EXPECT().CallerID().Return(mopts.callerID).AnyTimes()
		fw.EXPECT().Certname().DoAndReturn(func() string {
			if fw.Configuration().OverrideCertname != "" {
				return fw.Configuration().OverrideCertname
			}

			parts := strings.SplitN(mopts.callerID, "=", 2)
			return parts[1]
		}).AnyTimes()
	}

	return fw, fw.Configuration()
}
