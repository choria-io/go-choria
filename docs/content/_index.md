+++
weight = 5
+++

## Overview

Choria is a framework that can be used to build Control Planes, Orchestration Systems, Programmable Infrastructure or IoT networks.

This is a Golang Project hosting:

 * A Network Broker used for communications based on [NATS](https://nats.io)
 * A Server that deploys to every node and expose a API over the Choria Network Broker
 * A set of Client libraries and related tools to interact with the exposed APIs
 * [Autonomous Agent](https://choria.io/docs/autoagents/) framework used to create no-code control loops
 * Our JetStream based [Choria Streams](https://choria.io/docs/streams/) streaming data technology
 * Various foundational features for monitoring, provisioning, automation and more

General information about the project can be found in the project [documentation site](https://choria.io/docs/), this site
focus on a deep dive on the Server and Broker specifically.

## Scalability

The Architecture resembles a client-server architecture with very fast NATS middleware as a transport layer.

![Architecture](https://choria.io/docs/basic_client_server_overview.png)

A single Choria Broker deployed on a $40/month cloud VM can be used to manage 50,000 connected devices. The RPC layer is capable
of communicating with 10s of thousands of nodes in sub-second round trip times.

The system is a cellular, highly scalable and highly available. It can though be run comfortably on a Raspberry PI.

## Status

The project is under active development and in use by some of the largest infrastructures in the world.

Agents written 10 years ago for The Marionette Collective are still usable today, we place great value on stability, however
the project is under active development and new research and development continues on a daily basis.
