package io.choria.mcorpc.authpolicy

default allow = false

allow {
   input.agent = "ginkgo"
   input.action = "boop"
}

allow {
   input.agent = "other"
   input.action = "poob"
}