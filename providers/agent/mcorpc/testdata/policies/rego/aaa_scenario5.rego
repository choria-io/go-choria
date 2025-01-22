package io.choria.aaasvc

default allow = false

allow if {
    input.agent == "myco"
    input.action == "deploy"
    input.data.component == "frontend"
    requires_class_filter("apache")
    requires_identity_filter("some.node")
    requires_fact_filter("country=mt")
    input.collective == "ginkgo"
    input.ttl == 60
    input.sender == "some.node"
    input.site == "ginkgo"
    input.claims.callerid == "up=bob"
    input.claims.user_properties.group == "admins"
}
