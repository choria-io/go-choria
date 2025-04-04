package io.choria.mcorpc.authpolicy

default allow = false

allow if {
    # Only allow a matching list
    sort(input.classes) = ["alpha", "beta"]
    # Only allow if classes is defined
    input.classes[_] = "alpha"
    input.classes[_] = "beta"
}
