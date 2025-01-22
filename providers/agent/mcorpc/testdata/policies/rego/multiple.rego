package io.choria.mcorpc.authpolicy

default allow = false

allow if {
   input.agent = "ginkgo"
   input.action = "boop"
}

allow if {
   input.agent = "other"
   input.action = "poob"
}
