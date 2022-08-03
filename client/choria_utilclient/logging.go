// generated code; DO NOT EDIT

package choria_utilclient

func (c *ChoriaUtilClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *ChoriaUtilClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaUtilClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
