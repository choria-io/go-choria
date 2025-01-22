package io.choria.mcorpc.authpolicy

default allow = false

allow if {
    input.facts.stub == true
    input.facts.buts = "big"
}
