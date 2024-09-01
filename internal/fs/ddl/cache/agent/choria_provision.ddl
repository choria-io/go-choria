metadata :name        => "choria_provision",
         :description => "Choria Provisioner",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.29.4",
         :url         => "https://choria.io",
         :timeout     => 20


action "configure", :description => "Configure the Choria Server" do
  display :failed

  input :action_policies,
        :prompt      => "Action Policy Documents",
        :description => "Map of Action Policy documents indexed by file name",
        :type        => :hash,
        :optional    => true


  input :ca,
        :prompt      => "CA Bundle",
        :description => "PEM text block for the CA",
        :type        => :string,
        :validation  => '^-----BEGIN CERTIFICATE-----',
        :maxlength   => 20480,
        :optional    => true


  input :certificate,
        :prompt      => "Certificate",
        :description => "PEM text block for the certificate",
        :type        => :string,
        :validation  => '^-----BEGIN CERTIFICATE-----',
        :maxlength   => 10240,
        :optional    => true


  input :config,
        :prompt      => "Configuration",
        :description => "The configuration to apply to this node",
        :type        => :string,
        :validation  => '^{.+}$',
        :maxlength   => 2048,
        :optional    => false


  input :ecdh_public,
        :prompt      => "ECDH Public Key",
        :description => "Required when sending a private key",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 64,
        :optional    => true


  input :key,
        :prompt      => "PEM text block for the private key",
        :description => "A RSA private key",
        :type        => :string,
        :validation  => '-----BEGIN RSA PRIVATE KEY-----',
        :maxlength   => 10240,
        :optional    => true


  input :opa_policies,
        :prompt      => "Open Policy Agent Policy Documents",
        :description => "Map of Open Policy Agent Policy documents indexed by file name",
        :type        => :hash,
        :optional    => true


  input :server_jwt,
        :prompt      => "Server JWT",
        :description => "JWT file used to identify the server to the broker for ed25519 based authentication",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 2048,
        :optional    => true


  input :ssldir,
        :prompt      => "SSL Dir",
        :description => "Directory for storing the certificate in",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 500,
        :optional    => true


  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => true




  output :message,
         :description => "Status message from the Provisioner",
         :type        => "string",
         :display_as  => "Message"

end

action "gen25519", :description => "Generates a new ED25519 keypair" do
  display :always

  input :nonce,
        :prompt      => "Nonce",
        :description => "Single use token to be signed by the private key being generated",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 64,
        :optional    => false


  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :directory,
         :description => "The directory where server.key and server.pub is written to",
         :type        => "string",
         :display_as  => "Directory"

  output :public_key,
         :description => "The ED255519 public key hex encoded",
         :type        => "string",
         :display_as  => "Public Key"

  output :signature,
         :description => "The signature of the nonce made using the new private key, hex encoded",
         :type        => "string",
         :display_as  => "Signature"

end

action "gencsr", :description => "Request a CSR from the Choria Server" do
  display :always

  input :C,
        :prompt      => "Country",
        :description => "Country Code",
        :type        => :string,
        :validation  => '^[A-Z]{2}$',
        :maxlength   => 2,
        :optional    => true


  input :L,
        :prompt      => "Locality",
        :description => "Locality or municipality (such as city or town name)",
        :type        => :string,
        :validation  => '^[\w\s-]+$',
        :maxlength   => 50,
        :optional    => true


  input :O,
        :prompt      => "Organization",
        :description => "Organization",
        :type        => :string,
        :validation  => '^[\w\s-]+$',
        :maxlength   => 50,
        :optional    => true


  input :OU,
        :prompt      => "Organizational Unit",
        :description => "Organizational Unit",
        :type        => :string,
        :validation  => '^[\w\s-]+$',
        :maxlength   => 50,
        :optional    => true


  input :ST,
        :prompt      => "State",
        :description => "State",
        :type        => :string,
        :validation  => '^[\w\s-]+$',
        :maxlength   => 50,
        :optional    => true


  input :cn,
        :prompt      => "Common Name",
        :description => "The certificate Common Name to place in the CSR",
        :type        => :string,
        :validation  => '^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]).)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])$',
        :maxlength   => 80,
        :optional    => true


  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :csr,
         :description => "PEM text block for the CSR",
         :type        => "string",
         :display_as  => "CSR"

  output :public_key,
         :description => "PEM text block of the public key that made the CSR",
         :type        => "string",
         :display_as  => "Public Key"

  output :ssldir,
         :description => "SSL directory as determined by the server",
         :type        => "string",
         :display_as  => "SSL Dir"

end

action "jwt", :description => "Re-enable provision mode in a running Choria Server" do
  display :always

  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :ecdh_public,
         :description => "The ECDH public key for calculating shared secrets",
         :type        => "string",
         :display_as  => "ECDH Public Key"

  output :jwt,
         :description => "The contents of the JWT token",
         :type        => "string",
         :display_as  => "JWT Token"

end

action "reprovision", :description => "Reenable provision mode in a running Choria Server" do
  display :always

  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :message,
         :description => "Status message from the Provisioner",
         :type        => "string",
         :display_as  => "Message"

end

action "restart", :description => "Restart the Choria Server" do
  display :failed

  input :splay,
        :prompt      => "Splay time",
        :description => "The configuration to apply to this node",
        :type        => :number,
        :optional    => true


  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :message,
         :description => "Status message from the Provisioner",
         :type        => "string",
         :display_as  => "Message"

end

action "release_update", :description => "Performs an in-place binary update and restarts Choria" do
  display :always

  input :repository,
        :prompt      => "Repository URL",
        :description => "HTTP(S) server hosting the update repository",
        :type        => :string,
        :validation  => '^http(s*)://',
        :maxlength   => 512,
        :optional    => false


  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false


  input :version,
        :prompt      => "Version to update to",
        :description => "Package version to update to",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 32,
        :optional    => false




  output :message,
         :description => "Status message from the Provisioner",
         :type        => "string",
         :display_as  => "Message"

end

action "shutdown", :description => "Shut the Choria Server down cleanly" do
  display :failed

  input :token,
        :prompt      => "Token",
        :description => "Authentication token to pass to the server",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 128,
        :optional    => false




  output :message,
         :description => "Status message from the Provisioner",
         :type        => "string",
         :display_as  => "Message"

end

