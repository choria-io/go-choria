# go-protocol

[![CodeFactor](https://www.codefactor.io/repository/github/choria-io/go-protocol/badge)](https://www.codefactor.io/repository/github/choria-io/go-protocol)

This is a Golang implementation of the Choria protocol.  It does not implement any networking or transport, just the protocol parts.

This is in use by the go-choria project that builds a new `mcollective`like server, broker and eventually clients.

## Protocol Design

The messages in a Choria network are made of up many layers, for example should we wish to send `hello world` to a remote host we have to do quite a bit of work to get there:

```
( transport like NATS
  ( transport packet that travels over the transport
    ( security plugin that securely wraps the payload in SSL signatures etc
        ( choria message that has details about sender, filters, etc
            ( message body like `hello world` )
        )
    )
  )
)
```

So similar to how lets say REST lives in HTTP lives in TCP etc which allows the actual data to travel anything from high speed fibre to pigeons the Choria protocol allows similar flexibility in choosing a transport.

The middle most message is where the MCollective RPC system lives and like with HTTP, FTP etc multiple protocols can cohabit this network.

In Choria with NATS the above looks like this:

```
( NATS with TLS
    ( JSON encoded choria:transport:1
        ( JSON encoded choria:secure:request:1 or choria:secure:reply:1
            ( JSON encoded choria:request:1 or choria:reply:1
                ( payload for/from a given agent)
            )
        )
    )
)
```

The strings like `choria:request:1` means it's a V1 protocol `choria:request` message and maps to a constant like `protocol.RequestV1`.

The protocol also supports Federation which further complicates matters as in federated networks there are additional wrapping of packets going on - in practice it's just data copied into the above structure rather than more wrapping.

JSON Schemas for the whole version 1protocol [can be found in the repo](https://github.com/choria-io/go-protocol/tree/master/protocol/v1/schema), these schemas are used to validate every step of the way.


## Examples

Create a request and package it for transport:

```go
// a request to the agent test_agent sent from a machine called my.host.name and a user
// identifying itself as having a certificate rip.mcollective.  The message may live for
// 120 seconds and is targetted at a sub collective de_collective
request, _ := v1.NewRequest("test_agent", "my.host.name", "choria=rip.mcollective", 120, "unique_req_id", "de_collective")

// the payload the request will copy for us
request.SetMessage("hello world")

// at this point you can claim to be anyone with any cert, no validation is done yet,
// this follows at the security layer, this will assert that the cert you give does actually
// match rip.mcollective in name and it will sign the message and fingerprint it, this will
// fail if you are unable to present a certificate that match what you claimed above
srequest, _ := v1.NewSecureRequest(request, "path/to/pubcert.pem", "path/to/privatecert.pem")

// now this message is validated that you have a matching cert and it cannot be tampered with
// by anyone, we can turn it into a transport
trequest, _ := v1.NewTransportMessage("rip.mcollective")
trequest.SetRequestData(srequest)

// finally we can get the JSON data to send over the wire using whatever means we like
j, _ := trequest.JSON()
```

Now the JSON above gets sent to a node using any means you like, Choria uses NATS.  Decoding this is done as follows:

```go
// read the choria config
cfg, _ := choria.NewConfig(choria.UserConfig())

// j here is what was received over the wire, we now have our Choria transport
trequest, _ := v1.NewTransportFromJSON(j)

// we parse the transport as a request which gives us a secure request - and validates the sender is
// signed by our CA etc, validates it matches the allowed cert regexes and determines if its a super
// user request or not
srequest, _ := v1.NewSecureRequestFromTransport(trequest, "/path/to/ca.pem", "/path/to/ssl_cache", cfg.Choria.CertnameWhitelist, cfg.Choria.PrivilegedUsers, false)

// we now get a request and inside it is the payload
request, _ := v1.NewRequestFromSecureRequest(srequest)

// prints "hellow world"
fmt.Println(request.Message())
```

This is to be honest a bit verbose, the Choria framework in `go-choria` has a bunch of helpers to make this much easier for you and also support things like detecting protocol versions and doing the right thing for you so in reality you'd probably use it via those, but it's totally usable without as you see.

The same basic process is followed for replies, you just need to keep unwrapping the onion :)
