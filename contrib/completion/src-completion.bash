#!/bin/bash

function _src() {
    cur="${COMP_WORDS[COMP_CWORD]}"
    case "${COMP_WORDS[COMP_CWORD-1]}" in
        "src")
            subcmds=$(src help -q)
            COMPREPLY=($(compgen -W "${subcmds}" -- ${cur}))
            ;;
        tool)
            toolchains=$(src info toolchains -q)
            COMPREPLY=($(compgen -W "${toolchains}" -- ${cur}))
            ;;
        *)
            case "${COMP_WORDS[COMP_CWORD-2]}" in
                tool)
                    toolchain="${COMP_WORDS[COMP_CWORD-1]}"
                    tools=$(src info tools -q -common -toolchain="$toolchain")
                    COMPREPLY=($(compgen -W "${tools}" -- ${cur}))
                    ;;
            esac
    esac
    return 0
}

complete -F _src src
