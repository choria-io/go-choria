// generated code; DO NOT EDIT

package choria_provisionclient

func (c *ChoriaProvisionClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *ChoriaProvisionClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaProvisionClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
