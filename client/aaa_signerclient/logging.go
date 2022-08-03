// generated code; DO NOT EDIT

package aaa_signerclient

func (c *AaaSignerClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *AaaSignerClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *AaaSignerClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
