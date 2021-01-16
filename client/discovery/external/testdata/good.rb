#!/usr/bin/env ruby

require "json"
require "pp"

def write_output(output)
  File.open(ENV["CHORIA_EXTERNAL_REPLY"], "w") {|f|
    f.puts(output.to_json)
  }
  exit
end

if ENV["CHORIA_EXTERNAL_PROTOCOL"] != "io.choria.choria.discovery.v1.external_request"
  write_output({"error" => "invalid protocol"})
  exit
end

request = JSON.parse(File.read(ENV["CHORIA_EXTERNAL_REQUEST"]))
expected = {
  "$schema" => "https://choria.io/schemas/choria/discovery/v1/external_request.json",
  "protocol" => "io.choria.choria.discovery.v1.external_request",
  "filter" => {
    "fact" => [{"fact" => "country", "operator"=>"==","value"=>"mt"}],
    "cf_class"=>[],
    "agent" => ["rpcutil"],
    "compound" => [],
    "identity" => []
  },
  "collective" => "ginkgo",
  "timeout" => 2,
}

if request != expected
  write_output({"error" => "invalid filter received: "+request.pretty_inspect})
else
  write_output({"protocol" => "io.choria.choria.discovery.v1.external_reply", "nodes" => ["one","two"]})
end
