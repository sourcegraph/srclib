#!/bin/bash

function _src() {
    cur="${COMP_WORDS[COMP_CWORD]}"
    case "${COMP_WORDS[COMP_CWORD-1]}" in
        "src")
            comms="tool tools info help"
            COMPREPLY=($(compgen -W "${comms}" -- ${cur}))
            ;;
        tool)
            tools=$(src tools -q)
            COMPREPLY=($(compgen -W "${tools}" -- ${cur}))
            ;;
    esac
    return 0
}

complete -F _src src
