package io.choria.mcorpc.authpolicy

default allow = false

allow if {
    # Only allow a matching list
    # This is not really a good idea, since the ordering can depend on sorts
    sort(input.agents) = sort(["ginkgo", "buts_agent", "stub_agent"])
    # Only allow if classes is defined
    input.agents[_] = "stub_agent"
    input.agents[_] = "buts_agent"
}
