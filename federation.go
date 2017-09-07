package protocol

// CopyFederationData copies the Federation related data from one message to another
func CopyFederationData(from Federable, to Federable) {
	if !from.IsFederated() {
		to.SetUnfederated()
		return
	}

	if reply, ok := from.FederationReplyTo(); ok {
		to.SetFederationReplyTo(reply)
	}

	if req, ok := from.FederationRequestID(); ok {
		to.SetFederationRequestID(req)
	}

	if targets, ok := from.FederationTargets(); ok {
		to.SetFederationTargets(targets)
	}
}
