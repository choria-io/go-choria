metadata :name        => "choria_registry",
         :description => "Choria Registry Service",
         :author      => "rip@devco.net <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.24.0",
         :url         => "https://choria.io",
         :provider    => "golang",
         :service     => true,
         :timeout     => 2


action "ddl", :description => "Retrieve the DDL for a specific plugin" do
  display :always

  input :format,
        :prompt      => "Plugin Format",
        :description => "The result format the plugin should be retrieved in",
        :type        => :list,
        :default     => "json",
        :list        => ["ddl", "json"],
        :optional    => true


  input :name,
        :prompt      => "Plugin Name",
        :description => "The name of the plugin",
        :type        => :string,
        :validation  => :shellsafe,
        :maxlength   => 64,
        :optional    => false


  input :plugin_type,
        :prompt      => "Plugin Type",
        :description => "The type of plugin",
        :type        => :list,
        :default     => "agent",
        :list        => ["agent"],
        :optional    => false




  output :ddl,
         :description => "The plugin DDL in the requested format",
         :type        => "string",
         :display_as  => "DDL"

  output :name,
         :description => "The name of the plugin",
         :type        => "string",
         :display_as  => "Name"

  output :plugin_type,
         :description => "The type of plugin",
         :type        => "string",
         :display_as  => "Type"

  output :version,
         :description => "The version of the plugin",
         :type        => "string",
         :display_as  => "Version"

end

