// generated code; DO NOT EDIT

package choria_registryclient

func (c *ChoriaRegistryClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *ChoriaRegistryClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ChoriaRegistryClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
