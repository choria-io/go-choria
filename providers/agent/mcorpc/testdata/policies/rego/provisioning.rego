package io.choria.mcorpc.authpolicy

default allow = false

allow if {
    # Only allow a matching list
    input.provisionMode == true
    # Only allow if classes is defined
}
