metadata :name        => "choria_util",
         :description => "Choria Utilities",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.27.0",
         :url         => "https://choria.io",
         :timeout     => 2


action "info", :description => "Choria related information from the running Daemon and Middleware" do
  display :failed



  output :choria_version,
         :description => "Choria version",
         :type        => "string",
         :display_as  => "Choria Version"

  output :client_flavour,
         :description => "Middleware client library flavour",
         :type        => "string",
         :display_as  => "Middleware Client Flavour"

  output :client_options,
         :description => "Active Middleware client options",
         :type        => "hash",
         :display_as  => "Middleware Client Options"

  output :client_stats,
         :description => "Middleware client statistics",
         :type        => "hash",
         :display_as  => "Middleware Client Stats"

  output :client_version,
         :description => "Middleware client library version",
         :type        => "string",
         :display_as  => "Middleware Client Library Version"

  output :connected_server,
         :description => "Connected middleware server",
         :type        => "string",
         :display_as  => "Connected Broker"

  output :connector,
         :description => "Connector plugin",
         :type        => "string",
         :display_as  => "Connector"

  output :connector_tls,
         :description => "If the connector is running with TLS security enabled",
         :type        => "boolean",
         :display_as  => "Connector TLS"

  output :facter_command,
         :description => "Command used for Facter",
         :type        => "string",
         :display_as  => "Facter"

  output :facter_domain,
         :description => "Facter domain",
         :type        => "string",
         :display_as  => "Facter Domain"

  output :middleware_servers,
         :description => "Middleware Servers configured or discovered",
         :type        => "array",
         :display_as  => "Middleware"

  output :path,
         :description => "Active OS PATH",
         :type        => "string",
         :display_as  => "Path"

  output :secure_protocol,
         :description => "If the protocol is running with PKI security enabled",
         :type        => "boolean",
         :display_as  => "Protocol Secure"

  output :security,
         :description => "Security Provider plugin",
         :type        => "string",
         :display_as  => "Security Provider"

  output :srv_domain,
         :description => "Configured SRV domain",
         :type        => "string",
         :display_as  => "SRV Domain"

  output :using_srv,
         :description => "Indicates if SRV records are considered",
         :type        => "boolean",
         :display_as  => "SRV Used"

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

action "machine_state", :description => "Retrieves the current state of a specific Choria Autonomous Agent" do
  display :ok

  input :instance,
        :prompt      => "Instance ID",
        :description => "Machine Instance ID",
        :type        => :string,
        :validation  => '^.+-.+-.+-.+-.+$',
        :maxlength   => 36,
        :optional    => true


  input :name,
        :prompt      => "Name",
        :description => "Machine Name",
        :type        => :string,
        :validation  => '^[a-zA-Z][a-zA-Z0-9_-]+',
        :maxlength   => 128,
        :optional    => true


  input :path,
        :prompt      => "Path",
        :description => "Machine Path",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 512,
        :optional    => true




  output :available_transitions,
         :description => "The list of available transitions this autonomous agent can make",
         :type        => "array",
         :display_as  => "Available Transitions"

  output :current_state,
         :description => "The Choria Scout specific state for Scout checks",
         :display_as  => "Scout State"

  output :id,
         :description => "The unique running ID of the autonomous agent",
         :type        => "string",
         :display_as  => "ID"

  output :name,
         :description => "The name of the autonomous agent",
         :type        => "string",
         :display_as  => "Name"

  output :path,
         :description => "The location on disk where the autonomous agent is stored",
         :type        => "string",
         :display_as  => "Path"

  output :scout,
         :description => "True when this autonomous agent represents a Choria Scout Check",
         :type        => "boolean",
         :display_as  => "Scout Check"

  output :start_time,
         :description => "The time the autonomous agent was started in unix seconds",
         :type        => "string",
         :display_as  => "Started"

  output :state,
         :description => "The current state the agent is in",
         :type        => "string",
         :display_as  => "State"

  output :version,
         :description => "The version of the autonomous agent",
         :type        => "string",
         :display_as  => "Version"

  summarize do
    aggregate summary(:state)
    aggregate summary(:name)
    aggregate summary(:version)
  end
end

action "machine_states", :description => "States of the hosted Choria Autonomous Agents" do
  display :always



  output :machine_ids,
         :description => "List of running machine IDs",
         :type        => "array",
         :display_as  => "Machine IDs"

  output :machine_names,
         :description => "List of running machine names",
         :type        => "array",
         :display_as  => "Machine Names"

  output :states,
         :description => "Hash map of machine statusses indexed by machine ID",
         :type        => "hash",
         :display_as  => "Machine States"

  summarize do
    aggregate summary(:machine_names)
  end
end

action "machine_transition", :description => "Attempts to force a transition in a hosted Choria Autonomous Agent" do
  display :failed

  input :instance,
        :prompt      => "Instance ID",
        :description => "Machine Instance ID",
        :type        => :string,
        :validation  => '^.+-.+-.+-.+-.+$',
        :maxlength   => 36,
        :optional    => true


  input :name,
        :prompt      => "Name",
        :description => "Machine Name",
        :type        => :string,
        :validation  => '^[a-zA-Z][a-zA-Z0-9_-]+',
        :maxlength   => 128,
        :optional    => true


  input :path,
        :prompt      => "Path",
        :description => "Machine Path",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 512,
        :optional    => true


  input :transition,
        :prompt      => "Transition Name",
        :description => "The transition event to send to the machine",
        :type        => :string,
        :validation  => '^[a-zA-Z][a-zA-Z0-9_-]+$',
        :maxlength   => 128,
        :optional    => false


  input :version,
        :prompt      => "Version",
        :description => "Machine Version",
        :type        => :string,
        :validation  => '^\d+\.\d+\.\d+$',
        :maxlength   => 20,
        :optional    => true




  output :success,
         :description => "Indicates if the transition was successfully accepted",
         :type        => "boolean",
         :display_as  => "Accepted"

end

