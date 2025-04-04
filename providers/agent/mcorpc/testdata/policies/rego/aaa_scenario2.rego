package io.choria.aaasvc

default allow = false

allow if {
    input.agent == "myco"
    input.action == "deploy"
}
