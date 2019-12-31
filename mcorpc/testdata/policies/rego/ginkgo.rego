package choria.mcorpc.authpolicy

default allow = false

allow {
	input.agent = "ginkgo"
	input.action = "boop"
	input.callerID = "choria=ginkgo.mcollective"
	input.facts.stub == true
}