package io.choria.aaasvc

default allow = false

allow {
    input.agent == "myco"
    input.action == "deploy"
    input.data.component == "frontend"
}
