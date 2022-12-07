+++
title = "Choria Broker"
toc = true
weight = 10
pre = "<b>1. </b>"
+++

Choria Broker is a set of features that, in a fully decentralized setup, is the only central component Choria needs. It
is highly available, clustered and very high performance. The messaging layer is based on [NATS](https://nats.io) with 
a number of Choria specific additions.

A single Choria Broker can manage 50,000 devices on a low budget compute instance - though a cluster of at least 3 brokers 
is recommended for availability reasons.

### Features

 * Choria Brokers is the core message passing middleware, this is a managed NATS Core instance
 * [Choria Streams](https://choria.io/docs/streams/) is the data streaming solution used by various Choria components, this is a managed NATS JetStream instance
 * [Choria Federation Broker](https://choria.io/docs/federation/) connects entirely isolated Choria networks into a federated single network
 * [Choria Data Adapters](https://choria.io/docs/adapters/) to move data from Choria Broker to other technologies
 * A Choria specific authentication layer
