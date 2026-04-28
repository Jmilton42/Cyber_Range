package forge

import (
	"fmt"
	"os"
)

const bashCompletion = `# bash completion for forge
_forge_complete() {
    local cur prev words cword
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
    }

    local commands="init validate plan apply destroy status serve doctor config logs reload subnets import snapshot start stop migrate networks cost new plugins help version"
    local subnets_subs="list free reserve"
    local networks_subs="prune"
    local plugins_subs="list"

    # Replace the static command list with the runtime list (built-ins +
    # discovered plugins) when the user is on the very first word.
    if [[ "${COMP_CWORD}" -eq 1 ]]; then
        local dyn
        dyn="$(forge __complete commands 2>/dev/null)"
        if [[ -n "${dyn}" ]]; then
            commands="${dyn}"
        fi
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return
    fi

    case "${COMP_CWORD}" in
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
                plugins)
                    COMPREPLY=( $(compgen -W "${plugins_subs}" -- "${cur}") )
                    return
                    ;;
                destroy|snapshot|start|stop|import|reload|migrate|cost)
                    local projects
                    projects="$(forge __complete projects 2>/dev/null)"
                    COMPREPLY=( $(compgen -W "${projects}" -- "${cur}") )
                    return
                    ;;
                new)
                    if [[ "${cur}" == --template=* ]]; then
                        local tpls
                        tpls="$(forge __complete templates 2>/dev/null)"
                        COMPREPLY=( $(compgen -W "${tpls}" -P "--template=" -- "${cur#--template=}") )
                        return
                    fi
                    COMPREPLY=( $(compgen -W "--template --name --dir --reserve --no-reserve" -- "${cur}") )
                    return
                    ;;
            esac
            ;;
        3)
            case "${COMP_WORDS[1]}" in
                subnets)
                    case "${COMP_WORDS[2]}" in
                        free|reserve)
                            local projects
                            projects="$(forge __complete projects 2>/dev/null)"
                            COMPREPLY=( $(compgen -W "${projects}" -- "${cur}") )
                            return
                            ;;
                    esac
                    ;;
                migrate)
                    local nodes
                    nodes="$(forge __complete nodes 2>/dev/null)"
                    COMPREPLY=( $(compgen -W "${nodes}" -- "${cur}") )
                    return
                    ;;
            esac
            ;;
    esac

    # Generic --project value completion (for networks prune --project X, etc.)
    if [[ "${prev}" == "--project" || "${prev}" == "-project" ]]; then
        local projects
        projects="$(forge __complete projects 2>/dev/null)"
        COMPREPLY=( $(compgen -W "${projects}" -- "${cur}") )
        return
    fi

    COMPREPLY=()
}
complete -F _forge_complete forge
`

const zshCompletion = `#compdef forge
_forge() {
    local -a commands subnets_subs networks_subs plugins_subs
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
        'cost:Per-project resource breakdown (vCPU/RAM/disk)'
        'new:Scaffold a new project from a template'
        'plugins:List discovered forge-* plugins'
        'help:Show help'
        'version:Show forge version'
    )
    subnets_subs=('list' 'free' 'reserve')
    networks_subs=('prune')
    plugins_subs=('list')

    if (( CURRENT == 2 )); then
        # Merge built-ins with any plugins discovered at runtime.
        local -a dyn
        dyn=( ${(f)"$(forge __complete commands 2>/dev/null)"} )
        if (( ${#dyn} )); then
            _describe 'forge command' dyn
        else
            _describe 'forge command' commands
        fi
        return
    fi

    case "${words[2]}" in
        subnets)
            if (( CURRENT == 3 )); then
                _describe 'subnets subcommand' subnets_subs
            elif (( CURRENT == 4 )); then
                case "${words[3]}" in
                    free|reserve)
                        local -a projects
                        projects=( ${(f)"$(forge __complete projects 2>/dev/null)"} )
                        _describe 'project' projects
                        ;;
                esac
            fi
            ;;
        networks)
            if (( CURRENT == 3 )); then
                _describe 'networks subcommand' networks_subs
            fi
            ;;
        plugins)
            if (( CURRENT == 3 )); then
                _describe 'plugins subcommand' plugins_subs
            fi
            ;;
        destroy|snapshot|start|stop|import|reload|migrate|cost)
            if (( CURRENT == 3 )); then
                local -a projects
                projects=( ${(f)"$(forge __complete projects 2>/dev/null)"} )
                _describe 'project' projects
            elif (( CURRENT == 4 )) && [[ "${words[2]}" == "migrate" ]]; then
                local -a nodes
                nodes=( ${(f)"$(forge __complete nodes 2>/dev/null)"} )
                _describe 'cluster node' nodes
            fi
            ;;
        new)
            if (( CURRENT == 3 )); then
                _arguments \
                    '--template[Template id]:template:->tpl' \
                    '--name[Project name]:name:' \
                    '--dir[Target directory]:dir:_files -/' \
                    '--reserve[Allocate subnet up front]' \
                    '--no-reserve[Skip subnet allocation]'
                if [[ "${state}" == "tpl" ]]; then
                    local -a tpls
                    tpls=( ${(f)"$(forge __complete templates 2>/dev/null)"} )
                    _describe 'template' tpls
                fi
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
