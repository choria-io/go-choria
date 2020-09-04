package cmd

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

type completionCommand struct {
	command

	showZsh   bool
	showBash  bool
	list      string
	agent     string
	action    string
	zshScript string
}

func (e *completionCommand) Setup() error {
	e.cmd = cli.app.Command("completion", "Shell completion support").Hidden()
	e.cmd.Flag("zsh", "ZSH completion script").BoolVar(&e.showZsh)
	e.cmd.Flag("bash", "Bash completion script").Default("true").BoolVar(&e.showBash)
	e.cmd.Flag("list", "List various discovered items").EnumVar(&e.list, "agents", "actions", "inputs")
	e.cmd.Flag("agent", "Limit to a specific agent").StringVar(&e.agent)
	e.cmd.Flag("action", "Limit to a specific action").StringVar(&e.action)

	return nil
}

func (e *completionCommand) Configure() error {
	err = commonConfigure()
	if err != nil {
		cfg, err = config.NewDefaultConfig()
		if err != nil {
			return err
		}
		cfg.Choria.SecurityProvider = "file"
	}

	cfg.DisableSecurityProviderVerify = true

	e.zshScript = `#compdef _choria choria

zstyle ':completion::complete:choria-*' menu select=2

_choria() {
  local command=${words[2]}

  if [ "$command" = "req" ] || [ "$command" = "rpc" ]; then
    curcontext="${curcontext%:*:*}:choria-${command}"

    local -a clist

    if (( CURRENT == 3 )); then
      _call_program choria-list-agents choria completion --zsh --list agents | while read -A hline; do
        clist=($clist "${hline}")
      done

      _describe -t choria-agents "Choria Agents" clist
    elif (( CURRENT == 4 )); then
      _call_program choria-list-actions choria completion --zsh --list actions --agent=${words[3]} | while read -A hline; do
        clist=($clist "${hline}")
      done

      _describe -t choria-actions "${words[3]} Actions" clist

    elif (( CURRENT > 4 )); then
      _call_program choria-list-inputs choria completion --zsh --list inputs --action=${words[4]} --agent=${words[3]} | while read hline; do
        clist=($clist $hline)
      done

      _describe -t choria-inputs "${words[4]} Inputs" clist -S =
    fi

  else
    local -a cmdlist

    _call_program choria-list-commands choria --completion-bash "${words[@]:1:$CURRENT}" | xargs -n 1 echo | while read -A hline; do
      cmdlist=($cmdlist "${hline}")
    done

    curcontext="${curcontext%:*:*}:choria-commands"

    _describe -t choria-commands 'Choria Commands' cmdlist
  fi
}
`
	return err
}

func (e *completionCommand) Run(wg *sync.WaitGroup) error {
	defer wg.Done()

	switch e.list {
	case "agents":
		e.listAgents()

	case "actions":
		if e.agent == "" {
			return fmt.Errorf("please indicate an agent to list actions for")
		}

		e.listActions()

	case "inputs":
		if e.agent == "" {
			return fmt.Errorf("please indicate an agent to list inputs for")
		}

		if e.action == "" {
			return fmt.Errorf("please indicate an action to list inputs for")
		}

		e.listInputs()

	default:
		switch {
		case e.showZsh:
			fmt.Println(e.zshScript)

		case e.showBash:
			return fmt.Errorf("bash completion script generation is not complete")
		}
	}

	return nil
}

func (e *completionCommand) listInputs() {
	ddl, err := agent.Find(e.agent, cfg.LibDir)
	if err != nil {
		return
	}

	act, err := ddl.ActionInterface(e.action)
	if err != nil {
		return
	}

	found := []string{}

	for _, i := range act.InputNames() {
		input, _ := act.GetInput(i)

		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", i, input.Description))
		case e.showBash:
			found = append(found, i)
		}
	}

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func (e *completionCommand) listActions() {
	found := []string{}

	ddl, err := agent.Find(e.agent, cfg.LibDir)
	if err != nil {
		return
	}

	for _, act := range ddl.Actions {
		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", act.Name, act.Description))
		case e.showBash:
			found = append(found, act.Name)
		}
	}

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func (e *completionCommand) listAgents() {
	found := []string{}

	agent.EachFile(cfg.LibDir, func(name string, path string) bool {
		ddl, err := agent.New(path)
		if err != nil {
			return false
		}

		switch {
		case e.showZsh:
			found = append(found, fmt.Sprintf("%s:%s", ddl.Metadata.Name, ddl.Metadata.Description))
		case e.showBash:
			found = append(found, ddl.Metadata.Name)
		}

		return false
	})

	sort.Strings(found)
	fmt.Println(strings.Join(found, "\n"))
}

func init() {
	cli.commands = append(cli.commands, &completionCommand{})
}
