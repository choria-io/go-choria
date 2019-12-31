package choria.mcorpc.authpolicy

default allow = false

allow {
    # Only allow a matching list
    input.provisionMode == true
    # Only allow if classes is defined
}
