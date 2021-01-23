#!/usr/bin/env ruby

abort("invalid argument") unless ARGV[0] == "discover"
abort("invalid argument") unless ARGV[1] == "--test"
abort("request file not found") unless File.exist?(ARGV[2])
abort("reply file not found") unless File.exist?(ARGV[3])
abort("invalid protocol") unless ARGV[4] == "io.choria.choria.discovery.v1.external_request"

exec(File.join(File.dirname(__FILE__), "good.rb"))
