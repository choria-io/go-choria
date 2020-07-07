metadata :name        => "scout",
         :description => "Choria Scout Management API",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.0.1",
         :url         => "https://choria.io",
         :provider    => "golang",
         :timeout     => 5


action "checks", :description => "Obtain a list of checks and their current status" do
  display :ok



  output :checks,
         :description => "Details about each check",
         :type        => "array",
         :display_as  => "Checks"

end

action "resume", :description => "Resume active checking of one or more checks" do
  display :failed

  input :checks,
        :prompt      => "Checks",
        :description => "Check to resume, empty means all",
        :type        => :array,
        :optional    => true




  output :failed,
         :description => "List of checks that could not be resumed",
         :type        => "array",
         :display_as  => "Failed"

  output :skipped,
         :description => "List of checks that was skipped",
         :type        => "array",
         :display_as  => "Skipped"

  output :transitioned,
         :description => "List of checks that were resumed",
         :type        => "array",
         :display_as  => "Triggered"

end

action "maintenance", :description => "Pause checking of one or more checks" do
  display :failed

  input :checks,
        :prompt      => "Checks",
        :description => "Check to pause, empty means all",
        :type        => :array,
        :optional    => true




  output :failed,
         :description => "List of checks that could not be paused",
         :type        => "array",
         :display_as  => "Failed"

  output :skipped,
         :description => "List of checks that was skipped",
         :type        => "array",
         :display_as  => "Skipped"

  output :transitioned,
         :description => "List of checks that were paused",
         :type        => "array",
         :display_as  => "Triggered"

end

action "trigger", :description => "Force an immediate check of one or more checks" do
  display :failed

  input :checks,
        :prompt      => "Checks",
        :description => "Check to trigger, empty means all",
        :type        => :array,
        :optional    => true




  output :failed,
         :description => "List of checks that could not be triggered",
         :type        => "array",
         :display_as  => "Failed"

  output :skipped,
         :description => "List of checks that was skipped",
         :type        => "array",
         :display_as  => "Skipped"

  output :transitioned,
         :description => "List of checks that were triggered",
         :type        => "array",
         :display_as  => "Triggered"

end

