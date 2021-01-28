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

	showZsh    bool
	showBash   bool
	list       string
	agent      string
	action     string
	zshScript  string
	bashScript string
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
	e.bashScript = `_choria_bash_autocomplete() {
    local cur prev opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"

    if ( _array_contains COMP_WORDS "req" || _array_contains COMP_WORDS "rpc" ) && [[ ${COMP_WORDS[$COMP_CWORD]} != "-"* ]] ; then
        _choria_req_bash_autocomplete
    else
        opts=$( ${COMP_WORDS[0]} --completion-bash ${COMP_WORDS[@]:1:$COMP_CWORD} )
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    fi
    return 0
}

# https://stackoverflow.com/a/14367368
_array_contains() {
    local array="$1[@]"
    local seeking=$2
    local in=1
    for element in "${!array}"; do
        if [[ $element == "$seeking" ]]; then
            in=0
            break
        fi
    done
    return $in
}

_choria_req_bash_autocomplete() {
    # Assume for completion to work, the command line must match
    # choria [options] <req|rpc> [agent [action]]
    # Options are not alloed to appear between req and the action

    # Find the index of req/rpc in the input to serve as the anchor point for where the agent/action appear
    for index in "${!COMP_WORDS[@]}"; do
        if [[ "${COMP_WORDS[$index]}" = "req" ]] || [[ "${COMP_WORDS[$index]}" = "rpc" ]] ; then
            BASE_INDEX=$index
            break
        fi
    done

    AGENT_INDEX=$(expr $BASE_INDEX + 1)
    ACTION_INDEX=$(expr $BASE_INDEX + 2)

    # If the agent/action are already selected, present the inputs and long-options as further completions
    if [[ "${#COMP_WORDS[@]}" -gt $(expr $ACTION_INDEX + 1) ]] ; then
        local INPUTS=$(choria completion --list inputs --agent ${COMP_WORDS[$AGENT_INDEX]} --action ${COMP_WORDS[$ACTION_INDEX]} 2>/dev/null | sed -e 's/$/=/')

        # Prevent inputs from having a space added to them, since they need to be in KEY=VALUE format
        compopt -o nospace

        COMPREPLY=($(compgen -W "${INPUTS}" -- ${COMP_WORDS[$COMP_CWORD]}))

    # If the agent is selected, present the available actions on the selected agent as completions
    elif [[ "${#COMP_WORDS[@]}" -gt $(expr $AGENT_INDEX + 1) ]]; then
        local ACTIONS=$(choria completion --list actions --agent ${COMP_WORDS[$AGENT_INDEX]} 2>/dev/null)
        COMPREPLY=($(compgen -W "${ACTIONS}" -- ${COMP_WORDS[$COMP_CWORD]}))

    # If nothing is selected, present the available agents as completions
    elif [[ "${#COMP_WORDS[@]}" -gt $(expr $BASE_INDEX + 1) ]] ; then
        local AGENTS=$(choria completion --list agents 2>/dev/null)
        COMPREPLY=($(compgen -W "${AGENTS}" -- ${COMP_WORDS[$COMP_CWORD]}))
    fi
}

complete -F _choria_bash_autocomplete choria
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
			fmt.Println(e.bashScript)
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
