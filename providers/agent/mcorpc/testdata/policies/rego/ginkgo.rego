package io.choria.mcorpc.authpolicy

default allow = false

allow if {
	input.agent = "ginkgo"
	input.action = "boop"
	input.callerid = "choria=ginkgo.mcollective"
	input.facts.stub == true
}
