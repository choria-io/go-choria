metadata    :name        => "choria_util",
            :description => "Choria Utilities",
            :author      => "R.I.Pienaar <rip@devco.net>",
            :license     => "Apache-2.0",
            :version     => "0.19.0",
            :url         => "https://choria.io",
            :timeout     => 2

requires :mcollective => "2.9.0"

action "machine_transition", :description => "Attempts to force a transition in a hosted Choria Autonomous Agent" do
  input :instance,
        :prompt => "Instance ID",
        :description => "Machine Instance ID",
        :type => :string,
        :validation => '^.+-.+-.+-.+-.+$',
        :maxlength => 36,
        :optional => true

  input :version,
        :prompt => "Version",
        :description => "Machine Version",
        :type => :string,
        :validation => '^\d+\.\d+\.\d+$',
        :maxlength => 20,
        :optional => true

  input :name,
        :prompt => "Name",
        :description => "Machine Name",
        :type => :string,
        :validation => '^[a-zA-Z][a-zA-Z0-9_-]+',
        :maxlength => 128,
        :optional => true

  input :path,
        :prompt => "Path",
        :description => "Machine Path",
        :type => :string,
        :validation => '.+',
        :maxlength => 512,
        :optional => true

  input :transition,
        :prompt => "Transition Name",
        :description => "The transition event to send to the machine",
        :type => :string,
        :validation => '^[a-zA-Z][a-zA-Z0-9_-]+$',
        :maxlength => 128,
        :optional => false

  output :success,
         :description => "Indicates if the transition was successfully accepted",
         :display_as => "Accepted"
end

action "machine_states", :description => "States of the hosted Choria Autonomous Agents" do
  display :always

  output :machine_names,
         :description => "List of running machine names",
         :display_as => "Machine Names"

  output :machine_ids,
         :description => "List of running machine IDs",
         :display_as => "Machine IDs"

  output :states,
         :description => "Hash map of machine statusses indexed by machine ID",
         :display_as => "Machine States"

  summarize do
    aggregate summary(:machine_names)
  end
end

action "info", :description => "Choria related information from the running Daemon and Middleware" do
  output :security,
         :description => "Security Provider plugin",
         :display_as => "Security Provider"

  output :secure_protocol,
         :description => "If the protocol is running with PKI security enabled",
         :display_as => "Protocol Secure"

  output :connector,
         :description => "Connector plugin",
         :display_as => "Connector"

  output :connector_tls,
         :description => "If the connector is running with TLS security enabled",
         :display_as => "Connector TLS"

  output :path,
         :description => "Active OS PATH",
         :display_as => "Path"

  output :choria_version,
         :description => "Choria version",
         :display_as => "Choria Version"

  output :client_version,
         :description => "Middleware client library version",
         :display_as => "Middleware Client Library Version"

  output :client_flavour,
         :description => "Middleware client gem flavour",
         :display_as => "Middleware Client Flavour"

  output :client_options,
         :description => "Active Middleware client gem options",
         :display_as => "Middleware Client Options"

  output :connected_server,
         :description => "Connected middleware server",
         :display_as => "Connected Broker"

  output :client_stats,
         :description => "Middleware client gem statistics",
         :display_as => "Middleware Client Stats"

  output :facter_domain,
         :description => "Facter domain",
         :display_as => "Facter Domain"

  output :facter_command,
         :description => "Command used for Facter",
         :display_as => "Facter"

  output :srv_domain,
         :description => "Configured SRV domain",
         :display_as => "SRV Domain"

  output :using_srv,
         :description => "Indicates if SRV records are considered",
         :display_as => "SRV Used"

  output :middleware_servers,
         :description => "Middleware Servers configured or discovered",
         :display_as => "Middleware"

  summarize do
    aggregate summary(:choria_version)
    aggregate summary(:client_version)
    aggregate summary(:client_flavour)
    aggregate summary(:connected_server)
    aggregate summary(:srv_domain)
    aggregate summary(:using_srv)
    aggregate summary(:secure_protocol)
    aggregate summary(:connector_tls)
  end
end
