package client

import (
	"github.com/sirupsen/logrus"
)

type ScoutAPI struct {
	log   *logrus.Entry
	cfile string
}

func NewAPIClient(cfile string, log *logrus.Entry) (*ScoutAPI, error) {
	return &ScoutAPI{log, cfile}, nil
}
