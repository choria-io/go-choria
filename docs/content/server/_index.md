+++
title = "Choria Server"
toc = true
weight = 20
pre = "<b>2. </b>"
+++

Choria Server is the component that runs on every managed device. It hosts various plugins that can be accessed
remotely via a RPC layer.

It's a stable and robust agent designed to run forever with minimal resource overheads on managed devices.

### Features

 * Hosts Choria RPC Agents
 * Hosts [Choria Autonomous Agents](https://choria.io/docs/autoagents/)
 * Hosts foundational technology for [Choria Scout](https://choria.io/docs/scout/)
 * Supports optional self-provisioning and enrollment into a Choria network in a IoT device like manner
 * Communicates using a JSON based network protocol with extensive [JSON Schemas](https://github.com/choria-io/schemas/tree/master/choria)
 * Deep RBAC integration for Authentication, Authorization and Auditing
 * Supports mTLS or JWT token based security layers with, optional, integration into Enterprise SSO, IAM and systems like Hashicorp Vault  
 * Supports [Open Policy Agent](https://www.openpolicyagent.org/) for Authorization
 * Emits [Cloud Events](https://cloudevents.io/) for network management and observability
 * Embeddable in Go applications to provide in-process management for automation backplanes
 * Extensive features related to gathering and streaming of Node Metadata
 * Distributed as RPM, Deb, DMG, MSI across many architectures
