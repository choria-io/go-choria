// generated code; DO NOT EDIT

package executorclient

func (c *ExecutorClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *ExecutorClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *ExecutorClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
