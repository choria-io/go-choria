metadata :name        => "scout",
         :description => "Choria Scout Agent Management API",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.26.2",
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

action "goss_validate", :description => "Performs a Goss validation using a specific file" do
  display :failed

  input :file,
        :prompt      => "Goss File",
        :description => "Path to the Goss validation specification",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 256,
        :optional    => true


  input :vars,
        :prompt      => "Vars File",
        :description => "Path to a file to use as template variables",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 256,
        :optional    => true


  input :yaml_rules,
        :prompt      => "Gossfile contents",
        :description => "Contents of the Gossfile to validate",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 5120,
        :optional    => true


  input :yaml_vars,
        :prompt      => "Variables YAML",
        :description => "YAML data to use as variables",
        :type        => :string,
        :validation  => '.+',
        :maxlength   => 5120,
        :optional    => true




  output :failures,
         :description => "The number of tests that failed",
         :type        => "integer",
         :display_as  => "Failed Tests"

  output :results,
         :description => "The full test results",
         :type        => "array",
         :display_as  => "Results"

  output :runtime,
         :description => "The time it took to run the tests, in seconds",
         :type        => "integer",
         :display_as  => "Runtime"

  output :skipped,
         :description => "Indicates how many tests were skipped",
         :type        => "integer",
         :display_as  => "Skipped"

  output :success,
         :description => "Indicates how many tests passed",
         :type        => "integer",
         :display_as  => "Success"

  output :summary,
         :description => "A human friendly test result",
         :type        => "string",
         :display_as  => "Summary"

  output :tests,
         :description => "The number of tests that were run",
         :type        => "integer",
         :display_as  => "Tests"

  summarize do
    aggregate summary(:tests, :format => "%s Tests on %d node(s)")
    aggregate summary(:failures, :format => "%s Failed test on %d node(s)")
    aggregate summary(:success, :format => "%s Passed tests on %d node(s)")
  end
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

