#!/bin/bash

function _src() {
    cur="${COMP_WORDS[COMP_CWORD]}"
    case "${COMP_WORDS[COMP_CWORD-1]}" in
        "src")
            subcmds=$(src help -q)
            COMPREPLY=($(compgen -W "${subcmds}" -- ${cur}))
            ;;
        tool)
            tools=$(src info tools -q)
            COMPREPLY=($(compgen -W "${tools}" -- ${cur}))
            ;;
        *)
            case "${COMP_WORDS[COMP_CWORD-2]}" in
                tool)
                    tool="${COMP_WORDS[COMP_CWORD-1]}"
                    ops=$(src info ops -q -common -tool="$tool")
                    COMPREPLY=($(compgen -W "${ops}" -- ${cur}))
                    ;;
            esac
    esac
    return 0
}

complete -F _src src
