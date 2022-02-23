metadata :name        => "rpcutil",
         :description => "Utility actions that expose information about the state of the running Server",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.25.0",
         :url         => "https://choria.io/",
         :timeout     => 2


action "agent_inventory", :description => "Inventory of all agents on the server including versions, licenses and more" do
  display :always



  output :agents,
         :description => "List of agents on the server",
         :type        => "array",
         :display_as  => "Agents"

end

action "collective_info", :description => "Info about the main and sub collectives that the server belongs to" do
  display :always



  output :collectives,
         :description => "All Collectives",
         :type        => "array",
         :display_as  => "All Collectives"

  output :main_collective,
         :description => "The main Collective",
         :type        => "string",
         :display_as  => "Main Collective"

  summarize do
    aggregate summary(:collectives)
  end
end

action "daemon_stats", :description => "Get statistics from the running daemon" do
  display :always



  output :agents,
         :description => "List of agents loaded",
         :type        => "array",
         :display_as  => "Agents"

  output :configfile,
         :description => "Config file used to start the daemon",
         :type        => "string",
         :display_as  => "Config File"

  output :filtered,
         :description => "Count of message that didn't pass filter checks",
         :type        => "integer",
         :display_as  => "Failed Filter"

  output :passed,
         :description => "Count of messages that passed filter checks",
         :type        => "integer",
         :display_as  => "Passed Filter"

  output :pid,
         :description => "Process ID of the Choria Server",
         :type        => "integer",
         :display_as  => "PID"

  output :replies,
         :description => "Count of replies sent back to clients",
         :type        => "integer",
         :display_as  => "Replies"

  output :starttime,
         :description => "Time the Choria Server started in unix seconds",
         :type        => "integer",
         :display_as  => "Start Time"

  output :threads,
         :description => "List of threads active in the Choria Server",
         :type        => "array",
         :display_as  => "Threads"

  output :times,
         :description => "Processor time consumed by the Choria Server",
         :type        => "hash",
         :display_as  => "Times"

  output :total,
         :description => "Count of messages received by the Choria Server",
         :type        => "integer",
         :display_as  => "Total Messages"

  output :ttlexpired,
         :description => "Count of messages that did pass TTL checks",
         :type        => "integer",
         :display_as  => "TTL Expired"

  output :unvalidated,
         :description => "Count of messages that failed security validation",
         :type        => "integer",
         :display_as  => "Failed Security"

  output :validated,
         :description => "Count of messages that passed security validation",
         :type        => "integer",
         :display_as  => "Security Validated"

  output :version,
         :description => "Choria Server Version",
         :type        => "string",
         :display_as  => "Version"

  summarize do
    aggregate summary(:version)
    aggregate summary(:agents)
  end
end

action "get_config_item", :description => "Get the active value of a specific config property" do
  display :always

  input :item,
        :prompt      => "Configuration Item",
        :description => "The item to retrieve from the server",
        :type        => :string,
        :validation  => '^.+$',
        :maxlength   => 120,
        :optional    => false




  output :item,
         :description => "The config property being retrieved",
         :type        => "string",
         :display_as  => "Property"

  output :value,
         :description => "The value that is in use",
         :display_as  => "Value"

  summarize do
    aggregate summary(:value)
  end
end

action "get_data", :description => "Get data from a data plugin" do
  display :always

  input :query,
        :prompt      => "Query",
        :description => "The query argument to supply to the data plugin",
        :type        => :string,
        :validation  => '^.+$',
        :maxlength   => 200,
        :optional    => true


  input :source,
        :prompt      => "Data Source",
        :description => "The data plugin to retrieve information from",
        :type        => :string,
        :validation  => '^\w+$',
        :maxlength   => 50,
        :optional    => false




end

action "get_fact", :description => "Retrieve a single fact from the fact store" do
  display :always

  input :fact,
        :prompt      => "The name of the fact",
        :description => "The fact to retrieve",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 512,
        :optional    => false




  output :fact,
         :description => "The name of the fact being returned",
         :type        => "string",
         :display_as  => "Fact"

  output :value,
         :description => "The value of the fact",
         :display_as  => "Value"

  summarize do
    aggregate summary(:value)
  end
end

action "get_facts", :description => "Retrieve multiple facts from the fact store" do
  display :always

  input :facts,
        :prompt      => "Comma-separated list of facts to retrieve",
        :description => "Facts to retrieve",
        :type        => :string,
        :validation  => '^\s*[\w\.\-]+(\s*,\s*[\w\.\-]+)*$',
        :maxlength   => 200,
        :optional    => false




  output :values,
         :description => "List of values of the facts",
         :type        => "hash",
         :display_as  => "Values"

end

action "inventory", :description => "System Inventory" do
  display :always



  output :agents,
         :description => "List of agent names",
         :type        => "array",
         :display_as  => "Agents"

  output :classes,
         :description => "List of classes on the system",
         :type        => "array",
         :display_as  => "Classes"

  output :collectives,
         :description => "All Collectives",
         :type        => "array",
         :display_as  => "All Collectives"

  output :data_plugins,
         :description => "List of data plugin names",
         :type        => "array",
         :display_as  => "Data Plugins"

  output :facts,
         :description => "List of facts and values",
         :type        => "hash",
         :display_as  => "Facts"

  output :machines,
         :description => "Autonomous Agents",
         :type        => "hash",
         :display_as  => "Machines"

  output :main_collective,
         :description => "The main Collective",
         :type        => "string",
         :display_as  => "Main Collective"

  output :version,
         :description => "Choria Server Version",
         :type        => "string",
         :display_as  => "Version"

end

action "ping", :description => "Responds to requests for PING with PONG" do
  display :always



  output :pong,
         :description => "The local Unix timestamp",
         :type        => "string",
         :display_as  => "Timestamp"

end

