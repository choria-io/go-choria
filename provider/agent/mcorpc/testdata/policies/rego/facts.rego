package io.choria.mcorpc.authpolicy

default allow = false

allow {
    input.facts.stub == true
    input.facts.buts = "big"
}
