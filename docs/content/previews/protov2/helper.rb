#!/usr/bin/ruby

require "json"
require "yaml"
require "base64"
require "net/http"
require "openssl"

BROKERS = "nats://broker.example.net:4222"
ISSUER = "b3989a299278750427b00213693c2ca02146476a361667682446230842836da8"

def empty_reply
  {
    "defer" => false,
    "msg" => "",
    "certificate" => "",
    "ca" => "",
    "configuration" => {},
    "server_claims" => {}
  }
end

def parse_input
  input = STDIN.read
  request = JSON.parse(input)
  request["inventory"] = JSON.parse(request["inventory"])

  request
end

def validate!(request, reply)
  if request["identity"] && request["identity"].length == 0
    reply["msg"] = "No identity received in request"
    reply["defer"] = true
    return false
  end

  unless request["ed25519_pubkey"]
    reply["msg"] = "No ed15519 public key received"
    reply["defer"] = true
    return false
  end

  unless request["ed25519_pubkey"]
    reply["msg"] = "No ed15519 directory received"
    reply["defer"] = true
    return false
  end

  if request["ed25519_pubkey"]["directory"].length == 0
    reply["msg"] = "No ed15519 directory received"
    reply["defer"] = true
    return false
  end

  true
end

def publish_reply(reply)
  puts reply.to_json
end

def publish_reply!(reply)
  publish_reply(reply)
  exit
end

def set_config!(request, reply)
  reply["configuration"].merge!(
    "plugin.choria.middleware_hosts" => BROKERS,
    "plugin.security.issuer.choria.public" => ISSUER,
    "identity" => request["identity"],
    "loglevel" => "info",
    "plugin.choria.server.provision" => "false",
    "rpcauthorization" => "1",
    "rpcauthprovider" => "aaasvc",
    "plugin.security.issuer.names" => "choria",
    "plugin.security.provider" => "choria",
    "plugin.security.choria.token_file" => File.join(request["ed25519_pubkey"]["directory"], "server.jwt"),
    "plugin.security.choria.seed_file" => File.join(request["ed25519_pubkey"]["directory"], "server.seed")
  )

  reply["server_claims"].merge!(
    "exp" => 5*60*60*24*365,
    "permissions" => {
      "streams" => true
    }
  )
end

reply = empty_reply

begin
  request = parse_input

  File.open("/tmp/request.json", "w") {|f| f.write(request.to_json)}

  reply["msg"] = "Validating"
  unless validate!(request, reply)
    publish_reply!(reply)
  end

  reply["msg"] = "Config"
  set_config!(request, reply)

  reply["msg"] = "Done"
  publish_reply!(reply)
rescue SystemExit
rescue Exception
  reply["msg"] = "Unexpected failure during provisioning: %s: %s" % [$!.class, $!.to_s]
  reply["defer"] = true
  publish_reply!(reply)
end
