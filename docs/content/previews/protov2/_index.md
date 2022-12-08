+++
title = "V2 Protocol & Security"
toc = true
weight = 10
pre = "<b>1. </b>"
+++

This is a guide for early adopters who wish to test and study the [Version 2 Protocol and Security](/adr/001/) project.

{{% notice warning %}}
This is a *Hard Mode* guide that does everything manually and with no Configuration Management.
{{% /notice %}}

### Requirements

 * Choria Nightly
 * [Choria AAA Service](https://choria-io.github.io/aaasvc/) Nightly
 * [Choria Provisioner](https://choria-io.github.io/provisioner/) Nightly
 * Docker on Intel CPU

### Security Credentials

Security is by means of a ed25519 key that signs JWTs, some JWTs form a chain and can sign others. Regardless of the signer verification can be done using the public key associated with the Organization Issuer.

![Chain Issuers](/org-issuers.png)

The Organization Issuer can be kept offline with Provisioning and AAA being delegated authorities capable of signing servers and clients but these are optional components - the Organization Issuer can directly sign Clients and Servers allowing them to operate without the other central components.

### Deployment Methods

We demonstrate two deployment methods:

 * [Decentralized](#decentralized-deployment) - like traditional Choria with only a broker as shared component
 * Centralized AAA and Provisioning (TBD) - uses Choria AAA Service and Choria Provisioner for low-touch auto enrolment of Clients and Servers

Additionally we show how Hashicorp Vault can be integrated to manage the Organization Issuer

### Decentralized deployment

In this model we will deploy a system that resembles the basic architecture diagram below

![Architecture](https://choria.io/docs/basic_client_server_overview.png)

We have only the Brokers as central architecture with no Central AAA or Provisioning. 

{{% notice tip %}}
We will not use mTLS in this case. mTLS is supported but a major advantage of this mode is that it's not required.
{{% /notice %}}

#### Docker

We will need a Docker network and 3 instances - broker, server, client and issuer.

```nohighlight
$ docker network create choria_v2proto
$ docker network create choria_issuer
$ docker pull choria/choria:nightly
$ docker run -ti --rm --entrypoint bash \
      --network choria_v2proto \
      --hostname broker.example.net \
      choria/choria:nightly -l
$ docker run -ti --rm --entrypoint bash \
      --network choria_v2proto 
      --hostname server.example.net \
      choria/choria:nightly -l
$ docker run -ti --rm --entrypoint bash \
      --network choria_v2proto \
      --hostname client.example.net \
      choria/choria:nightly -l
$ docker run -ti --rm --entrypoint bash \
      --network choria_issuer \
      --hostname issuer.example.net choria/choria:nightly -l
```

The issuer is not needed per-se but will demonstrate that the Issuer credentials never need to near the managed network.

#### Keys and JWTs

In this Scenario we need:

 * An Organization Issuer keypair
 * A client JWT for each user
 * A server JWT for each server
 * x509 certificate for the broker TLS port

In this scenario you are responsible for creating and distributing the keys and tokens.

##### Organization Issuer

The Organization Issuer is the root of the Trust Chain and is a ed25519 key, let's create the keypair on the issuer node:

```nohighlight
[choria@issuer ~]$ mkdir -p development/issuer
[choria@issuer ~]$ choria jwt keys development/issuer/private.key development/issuer/public.key
Public Key: b3989a299278750427b00213693c2ca02146476a361667682446230842836da8

Ed25519 seed saved in development/issuer/private.key
```

{{% notice warning %}}
This key should be kept private and ideally in a Hashicorp Vault server. See a later section for guidance on Vault.
{{% /notice %}}

##### Broker JWT and Config

Every broker needs a ed25519 Keypair and a signed JWT.

First we create a keypair on the broker, the private key never leaves the broker:

```nohighlight
[choria@broker ~]$ choria jwt keys /etc/choria/private.key /etc/choria/public.key
Public Key: 8918c1c7a4aeb4d4ad16729dc9b9c12df021d9296106eb5f072b224aa8f8eee9

Ed25519 seed saved in /etc/choria/private.key
```

Pass the Public Key to the Organization Issuer who creates a JWT:

```nohighlight
[choria@issuer ~]$ mkdir -p development/broker/broker.example.net
[choria@issuer ~]$ choria jwt server development/broker/broker.example.net/token.jwt \
server.example.net \
8918c1c7a4aeb4d4ad16729dc9b9c12df021d9296106eb5f072b224aa8f8eee9 \
development/issuer/private.key \
--collectives=choria
```

With access to just the Broker public key the Organization Issuer can create a server token, pass this back to the server who stores it in `/etc/choria/broker.jwt`.

{{% notice tip %}}
Note that for version 2 protocol the default collective is `["choria"]`.
{{% /notice %}}

```nohighlight
[choria@issuer ~]$ choria jwt development/broker/broker.example.net/token.jwt development/issuer/public.key
Validated Server Token development/server/server.example.net/token.jwt

             Identity: server.example.net
           Expires At: 2023-12-08 13:03:23 +0000 UTC (364d23h59m41s)
          Collectives: choria
           Public Key: 8918c1c7a4aeb4d4ad16729dc9b9c12df021d9296106eb5f072b224aa8f8eee9
    Organization Unit: choria
   Private Network ID: 92328d88bef9d063480fd4b0ec5e4879

   Broker Permissions:

          No server specific permissions granted
```

We pass the JWT back to the broker and save in `/etc/choria/broker.jwt`.

The broker need x509 certificates to open the TLS network port, here we just self-sign one but you can get those from anywhere.

```nohighlight
[choria@broker ~]$ openssl genrsa -out /etc/choria/broker-tls.key 2048
Generating RSA private key, 2048 bit long modulus (2 primes)
..+++++
....................................................................................................+++++
e is 65537 (0x010001)
[choria@broker ~]$ openssl req -new -x509 -sha256 -key /etc/choria/broker-tls.key \
   -out /etc/choria/broker-tls.cert -days 365 -subj "/O=Choria.io/CN=broker.example.net"
```

With all in place it should look like this:

```nohighlight
[choria@broker ~]$ find /etc/choria/
/etc/choria/
/etc/choria/broker.conf
/etc/choria/broker-tls.key
/etc/choria/broker-tls.cert
/etc/choria/private.key
/etc/choria/public.key
/etc/choria/broker.jwt
```

We create the broker configuration in `/etc/choria/broker.conf` and start it, you need to change your issuer here:

```nohighlight
# The name of the organization to configure, for now only supports choria
plugin.security.issuer.names = choria

# The public key from the issuer
plugin.security.issuer.choria.public = b3989a299278750427b00213693c2ca02146476a361667682446230842836da8

plugin.choria.network.system.password = sYst3m
plugin.choria.stats_port = 8222
plugin.choria.broker_network = true
plugin.choria.network.client_port = 4222
plugin.choria.network.stream.store = /data
plugin.choria.network.system.user = system
loglevel = info
plugin.choria.use_srv = false

plugin.security.provider = choria
plugin.security.choria.certificate = /etc/choria/broker-tls.cert
plugin.security.choria.key = /etc/choria/broker-tls.key
plugin.security.choria.token_file = /etc/choria/broker.jwt
plugin.security.choria.seed_file = /etc/choria/private.key
```

Let's start the broker, showing the key lines from the output here:

```nohighlight
$ choria broker run --config /etc/choria/broker.conf
INFO[0000] Choria Broker version 0.99.0.20221201 starting with config /etc/choria/broker.conf
INFO[0000] Starting Network Broker
WARN[0000] Allowing unverified TLS connections for Organization Issuer issued connections  component=network
WARN[0000] Loaded Organization Issuer choria with public key b3989a299278750427b00213693c2ca02146476a361667682446230842836da8  component=network
INFO[0000] Listening for client connections on [::]:4222  component=network_broker
...
```

##### Server JWT

Every server needs a ed25519 Keypair and a signed JWT.

The server process is identical to the broker process except change `broker.example.net` to `server.example.net` in identities and make obvious file name changes. Servers do not need any x509 certificates like brokers.

Now we can configure and start the server, place this in `/etc/choria/server.conf`:

```nohighlight
# The name of the organization to configure, for now only supports choria
plugin.security.issuer.names = choria

# The public key from the issuer
plugin.security.issuer.choria.public = b3989a299278750427b00213693c2ca02146476a361667682446230842836da8

# We enable authorization and set it to trust the JWT tokens policy
rpcauthorization = 1
rpcauthprovider = aaasvc

plugin.security.provider = choria
plugin.security.choria.token_file = /etc/choria/server.jwt
plugin.security.choria.seed_file = /etc/choria/private.key
plugin.choria.middleware_hosts = nats://broker.example.net:4222
```

And finally let's run the server, showing key log lines only:

{{% notice tip %}}
Servers usually run as root, here as the `choria` user as it's in the container
{{% /notice %}}

```nohighlight
[choria@server ~]$ choria server run --config /etc/choria/server.conf
INFO[0000] Choria Server version 0.99.0.20221201 starting with config /etc/choria/server.conf using protocol version 2
INFO[0000] Setting JWT token and unique reply queues based on JWT for "server.example.net"  component=server connection=server.example.net identity=server.example.net
INFO[0000] Setting custom inbox prefix based on unique ID to choria.reply.77e64440ac709c0836487e5b77334e5b  component=server connection=server.example.net identity=server.example.net
```

###### Client JWT

Every client needs a ed25519 keypair and a signed JWT.

We will create a client that has access to Choria Streams and the ability to manage the fleet without any AAA Server.

The client will create their own keypair, so we run that in the client node:

```noghighlight
[choria@client ~]$ mkdir -p ~/.config/choria/
[choria@client ~]$ choria jwt keys ~/.config/choria/private.key ~/.config/choria/public.key
Public Key: 4bbfddb9f70f4b39f5b13bac8e83a9a31c3af49e388da86a666f8615101bc818

Ed25519 seed saved in /home/choria/.config/choria/private.key
```

This client `private.key` should be kept private and not shared, the JWT can be created with knowledge of the public key only.

The client pass their Public Key to the Organization Issuer who creates a JWT on the Issuer node:

{{% notice tip %}}
Here we use `choria` as the identity, this would match the unix user name.

If a user is on many machines, create a JWT per machine.
{{% /notice %}}

```nohighlight
[choria@issuer ~]$ mkdir -p development/client/choria
[choria@issuer ~]$ choria jwt client development/client/choria/token.jwt choria development/issuer/private.key \
 --public-key 4bbfddb9f70f4b39f5b13bac8e83a9a31c3af49e388da86a666f8615101bc818 \
 --stream-admin \
 --event-viewer \
 --elections-user \
 --service \
 --fleet-management \
 --agents '*' \
 --validity 1y
Saved token to development/client/choria/token.jwt, use 'choria jwt view development/client/choria/token.jwt' to view it
```

With access to the Issuer private key, but not the user private key, we can create a JWT for the user. Since we have no AAA Service we mark this user as a `service` which allows them to have a long token validity. We set a policy allowing all agent access, in real life this would be an Open Policy Agent policy.

```nohighlight
[choria@issuer ~]$ choria jwt development/client/choria/token.jwt development/issuer/public.key
Validated Client Identification Token development/client/choria/token.jwt

          Caller ID: choria
  Organization Unit: choria
     Allowed Agents: *
         Public Key: 4bbfddb9f70f4b39f5b13bac8e83a9a31c3af49e388da86a666f8615101bc818
 Private Network ID: 0a63c70a8817f5ef4d19d055ce6513f1
         Expires At: 2023-12-08 12:54:07 +0000 UTC (364d23h59m19s)

 Client Permissions:

      Can manage Choria fleet nodes
      Can use Leader Elections
      Can view Lifecycle and Autonomous Agent events
      Can administer Choria Streams
      Can access the Broker system account
      Can have an extended token lifetime
```

Pass the JWT back to the client who saves it in `~/.config/choria/token.jwt`.

```nohighlight
[choria@client ~]$ find ~/.config
/home/choria/.config
/home/choria/.config/choria
/home/choria/.config/choria/private.key
/home/choria/.config/choria/public.key
/home/choria/.config/choria/token.jwt
```

We create a system-wide client configuration in `/etc/choria/client.conf`:

```nohighlight
loglevel = warn
plugin.choria.middleware_hosts = broker.example.net:4222
plugin.choria.network.system.user = system
plugin.choria.network.system.password = sYst3m

plugin.security.provider = choria
plugin.security.choria.token_file = ~/.config/choria/token.jwt
plugin.security.choria.seed_file = ~/.config/choria/private.key
```

We can now test the client:

```nohighlight
[choria@client ~]$ choria ping
server.example.net                       time=3 ms

---- ping statistics ----
1 replies max: 4ms min: 4ms avg: 4ms overhead: 12ms
```

Other commands like `choria req choria_util info` should work demonstrating authorization works and `choria broker server list` should list the broker indicating Broker System Account access works. After a minute or so `choria broker stream ls` will show a list of Streams demonstrating Choria Streams authority worked.
