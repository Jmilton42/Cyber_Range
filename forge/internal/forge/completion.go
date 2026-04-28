package forge

import (
	"fmt"
	"os"
)

const bashCompletion = `# bash completion for forge
_forge_complete() {
    local cur prev words cword
    _init_completion || return

    local commands="init validate plan apply destroy status serve doctor config logs reload subnets import snapshot start stop migrate networks help version"
    local subnets_subs="list free reserve"
    local networks_subs="prune"

    case "${COMP_CWORDS}" in
        1)
            COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
            return
            ;;
        2)
            case "${COMP_WORDS[1]}" in
                subnets)
                    COMPREPLY=( $(compgen -W "${subnets_subs}" -- "${cur}") )
                    return
                    ;;
                networks)
                    COMPREPLY=( $(compgen -W "${networks_subs}" -- "${cur}") )
                    return
                    ;;
            esac
            ;;
    esac

    COMPREPLY=()
}
complete -F _forge_complete forge
`

const zshCompletion = `#compdef forge
_forge() {
    local -a commands subnets_subs networks_subs
    commands=(
        'init:Prepare working directory'
        'validate:Validate the configuration'
        'plan:Show planned changes'
        'apply:Create or update infrastructure'
        'destroy:Tear down infrastructure'
        'status:Show subnet allocation'
        'serve:Run the HTTP config server'
        'doctor:Run preflight checks'
        'config:Show resolved configuration'
        'logs:Tail server.log'
        'reload:POST /reload to running server'
        'subnets:Manage subnet allocations'
        'import:Register an existing LXD project'
        'snapshot:Snapshot every VM in a project'
        'start:Start every VM in a project'
        'stop:Stop every VM in a project'
        'migrate:Move VMs to a different cluster member'
        'networks:Manage OVN networks'
        'help:Show help'
        'version:Show forge version'
    )
    subnets_subs=('list' 'free' 'reserve')
    networks_subs=('prune')

    if (( CURRENT == 2 )); then
        _describe 'forge command' commands
        return
    fi

    case "${words[2]}" in
        subnets)
            if (( CURRENT == 3 )); then
                _describe 'subnets subcommand' subnets_subs
            fi
            ;;
        networks)
            if (( CURRENT == 3 )); then
                _describe 'networks subcommand' networks_subs
            fi
            ;;
    esac
}
_forge "$@"
`

// RunCompletion writes the requested shell completion script to stdout.
// Returns an error for unsupported shells so the caller can exit non-zero.
func RunCompletion(shell string) error {
	switch shell {
	case "bash":
		_, err := fmt.Fprint(os.Stdout, bashCompletion)
		return err
	case "zsh":
		_, err := fmt.Fprint(os.Stdout, zshCompletion)
		return err
	default:
		return fmt.Errorf("unsupported shell %q (supported: bash, zsh)", shell)
	}
}
