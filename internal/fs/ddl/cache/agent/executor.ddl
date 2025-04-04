metadata :name        => "executor",
         :description => "Choria Process Executor Management",
         :author      => "R.I.Pienaar <rip@devco.net>",
         :license     => "Apache-2.0",
         :version     => "0.29.4",
         :url         => "https://choria.io",
         :provider    => "golang",
         :timeout     => 20


action "signal", :description => "Sends a signal to a process" do
  display :always

  input :id,
        :prompt      => "Job ID",
        :description => "The unique ID for the job",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 20,
        :optional    => false


  input :signal,
        :prompt      => "Signal",
        :description => "The signal to send",
        :type        => :integer,
        :optional    => false




  output :pid,
         :description => "The PID that was signaled",
         :type        => "integer",
         :display_as  => "PID"

  output :running,
         :description => "If the process was running after signaling",
         :type        => "boolean",
         :display_as  => "Running"

end

action "list", :description => "Lists jobs matching certain criteria" do
  display :always

  input :action,
        :prompt      => "Action",
        :description => "The action that created a job",
        :type        => :string,
        :validation  => '^[\w]+$',
        :maxlength   => 20,
        :optional    => true


  input :agent,
        :prompt      => "Agent",
        :description => "The agent that create a job",
        :type        => :string,
        :validation  => '^[\w]+$',
        :maxlength   => 20,
        :optional    => true


  input :before,
        :prompt      => "Before",
        :description => "Unix timestamp to limit jobs on",
        :type        => :integer,
        :optional    => true


  input :caller,
        :prompt      => "Caller",
        :description => "The caller id that created a job",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 50,
        :optional    => true


  input :command,
        :prompt      => "Command",
        :description => "The command that was executed",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 256,
        :optional    => true


  input :completed,
        :prompt      => "Completed",
        :description => "Limit to jobs that were completed",
        :type        => :boolean,
        :optional    => true


  input :identity,
        :prompt      => "Identity",
        :description => "The host identity that created the job",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 256,
        :optional    => true


  input :requestid,
        :prompt      => "Request",
        :description => "The Request ID that created the job",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 20,
        :optional    => true


  input :running,
        :prompt      => "Running",
        :description => "Limits to running jobs",
        :type        => :boolean,
        :optional    => true


  input :since,
        :prompt      => "Since",
        :description => "Unix timestamp to limit jobs on",
        :type        => :integer,
        :optional    => true




  output :jobs,
         :description => "List of matched jobs",
         :type        => "hash",
         :display_as  => "Jobs"

end

action "status", :description => "Requests the status of a job by ID" do
  display :always

  input :id,
        :prompt      => "Job ID",
        :description => "The unique ID for the job",
        :type        => :string,
        :validation  => '.',
        :maxlength   => 20,
        :optional    => false




  output :action,
         :description => "The RPC Action that started the process",
         :type        => "string",
         :display_as  => "Action"

  output :agent,
         :description => "The RPC Agent that started the process",
         :type        => "string",
         :display_as  => "Agent"

  output :args,
         :description => "The command arguments, if the caller has access",
         :type        => "string",
         :display_as  => "Arguments"

  output :caller,
         :description => "The Caller ID who started the process",
         :type        => "string",
         :display_as  => "Caller"

  output :command,
         :description => "The command being executed, if the caller has access",
         :type        => "string",
         :display_as  => "Command"

  output :exit_code,
         :description => "The exit code the process terminated with",
         :type        => "integer",
         :display_as  => "Exit Code"

  output :exit_reason,
         :description => "If the process failed, the reason for th failure",
         :type        => "string",
         :display_as  => "Exit Reason"

  output :pid,
         :description => "The OS Process ID",
         :type        => "integer",
         :display_as  => "Pid"

  output :requestid,
         :description => "The Request ID that started the process",
         :type        => "string",
         :display_as  => "Request ID"

  output :running,
         :description => "Indicates if the process is still running",
         :type        => "boolean",
         :display_as  => "Running"

  output :start_time,
         :description => "Time that the process started",
         :type        => "string",
         :display_as  => "Started"

  output :started,
         :description => "Indicates if the process was started",
         :type        => "boolean",
         :display_as  => "Started"

  output :stderr_bytes,
         :description => "The number of bytes of STDERR output available",
         :type        => "integer",
         :display_as  => "STDERR Bytes"

  output :stdout_bytes,
         :description => "The number of bytes of STDOUT output available",
         :type        => "integer",
         :display_as  => "STDOUT Bytes"

  output :terminate_time,
         :description => "Time that the process terminated",
         :type        => "string",
         :display_as  => "Terminated"

end

