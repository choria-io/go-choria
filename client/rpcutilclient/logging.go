// generated code; DO NOT EDIT

package rpcutilclient

func (c *RpcutilClient) debugf(msg string, a ...any) {
	c.clientOpts.logger.Debugf(msg, a...)
}

func (c *RpcutilClient) infof(msg string, a ...any) {
	c.clientOpts.logger.Infof(msg, a...)
}

func (c *RpcutilClient) errorf(msg string, a ...any) {
	c.clientOpts.logger.Errorf(msg, a...)
}
