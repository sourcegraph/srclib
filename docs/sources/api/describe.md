# `src api describe`

The `src api describe` command is used by editor plugins to retrieve information about the identifier at a specific position in a file.

## Usage
[[.run src api describe -h]]

## Output
The output is defined in [api_cmds.go](https://github.com/sourcegraph/srclib/blob/e5295dfcd719535ff9cbb37a2771337d44fe5953/src/api_cmds.go#L190-L193), as a json representation of the following struct.  

The Def and Example structs are defined as follows in the Sourcegraph API.

[[.code "https://raw.githubusercontent.com/sourcegraph/go-sourcegraph/6937daba84bf2d0f919191fd74e5193171b4f5d5/sourcegraph/defs.go" 105 113]]

[[.code "https://raw.githubusercontent.com/sourcegraph/go-sourcegraph/6937daba84bf2d0f919191fd74e5193171b4f5d5/sourcegraph/defs.go" 236 252]]
