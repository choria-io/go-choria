// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package imock

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brutella/hc/util"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

type fwMockOpts struct {
	callerID    string
	logDiscard  bool
	cfg         *config.Config
	ddlResolver inter.DDLResolver
	ddls        [][3]string
	reqProto    protocol.ProtocolVersion
}
type fwMockOption func(*fwMockOpts)

func WithRequestProtocol(p protocol.ProtocolVersion) fwMockOption {
	return func(o *fwMockOpts) {
		o.reqProto = p
	}
}

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

func WithDDLResolver(r inter.DDLResolver) fwMockOption {
	return func(o *fwMockOpts) {
		o.ddlResolver = r
	}
}

func WithDDLFiles(kind string, plugin string, path string) fwMockOption {
	return func(o *fwMockOpts) {
		o.ddls = append(o.ddls, [3]string{kind, plugin, path})
	}
}

func NewFrameworkForTests(ctrl *gomock.Controller, logWriter io.Writer, opts ...fwMockOption) (*MockFramework, *config.Config) {
	mopts := &fwMockOpts{
		cfg:      config.NewConfigForTests(),
		reqProto: protocol.RequestV1,
	}
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

	if mopts.ddlResolver == nil {
		resolver := NewMockDDLResolver(ctrl)
		mopts.ddlResolver = resolver
		for _, ddl := range mopts.ddls {
			f, err := os.ReadFile(ddl[2])
			if err != nil {
				panic(fmt.Sprintf("ddl file %s: %s", ddl[2], err))
			}
			resolver.EXPECT().DDLBytes(gomock.Any(), gomock.Eq("agent"), gomock.Eq("package"), gomock.Any()).Return(f, nil).AnyTimes()
		}
	}

	fw.EXPECT().DDLResolvers().Return([]inter.DDLResolver{mopts.ddlResolver}, nil).AnyTimes()
	fw.EXPECT().RequestProtocol().Return(mopts.reqProto).AnyTimes()

	return fw, fw.Configuration()
}
