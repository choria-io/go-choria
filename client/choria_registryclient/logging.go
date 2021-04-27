// generated code; DO NOT EDIT

package choria_registryclient

func (c *ChoriaRegistryClient) infof(msg string, a ...interface{}) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaRegistryClient) errorf(msg string, a ...interface{}) {
	c.clientOpts.logger.Errorf(msg, a...)
}
