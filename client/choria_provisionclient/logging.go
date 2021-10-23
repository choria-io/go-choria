// generated code; DO NOT EDIT

package choria_provisionclient

func (c *ChoriaProvisionClient) infof(msg string, a ...interface{}) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaProvisionClient) errorf(msg string, a ...interface{}) {
	c.clientOpts.logger.Errorf(msg, a...)
}
