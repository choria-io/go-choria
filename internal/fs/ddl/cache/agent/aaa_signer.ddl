metadata :name        => "aaa_signer",
         :description => "Request Signer for Choria AAA Service",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.24.0",
         :url         => "https://github.com/choria-io/aaasvc",
         :provider    => "golang",
         :service     => true,
         :timeout     => 10


action "sign", :description => "Signs a RPC Request on behalf of a user" do
  display :always

  input :request,
        :prompt      => "RPC Request",
        :description => "The request to sign",
        :type        => :string,
        :validation  => :shellsafe,
        :maxlength   => 100240,
        :optional    => false


  input :token,
        :prompt      => "JWT Token",
        :description => "The JWT token authenticating the user",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 10024,
        :optional    => false




  output :secure_request,
         :description => "The signed Secure Request",
         :type        => "string",
         :display_as  => "Secure Request"

end

