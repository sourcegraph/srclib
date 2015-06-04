page_title: Overview

# Overview
<div class="alert alert-danger" role="alert">Note: The API is still in flux, and may change throughout the duration of this beta.</div>

The srclib API will be used through the invocation of subcommands of the `srclib` executable.

## API Commands

API commands return their responses as JSON, to facilitate the building of tools on top of srclib. Sourcegraph's [plugins](#TODO-plugins-overview) all make heavy use of the API commands.

<div class="alert alert-danger" role="alert">Note: The docs currently only show the Go representation of the output. See <a href="https://blog.golang.org/json-and-go">this blog post</a> for a primer on how Go types are marshaled into JSON.</div>

<!-- TODO: This should be generated from 'commands' in mkdocs.yml -->

### `srclib api describe`

[[.doc "cli/api_cmds.go" "APIDescribeCmdDoc"]]

#### Usage

[[.run srclib api describe -h]]

#### Output

[[.doc "cli/api_cmds.go" "APIDescribeCmdOutput"]]

### `srclib api list`
[[.doc "cli/api_cmds.go" "APIListCmdDoc"]]

#### Usage
[[.run srclib api list -h]]

#### Output
[[.doc "cli/api_cmds.go" "APIListCmdOutput"]]

### `srclib api deps`
[[.doc "cli/api_cmds.go" "APIDepsCmdDoc"]]

#### Usage
[[.run srclib api list -h]]

#### Output
[[.doc "cli/api_cmds.go" "APIDepsCmdOutput"]]

### `srclib api units`
[[.doc "cli/api_cmds.go" "APIUnitsCmdDoc"]]

#### Usage
[[.run srclib api units -h]]

#### Output
[[.doc "cli/api_cmds.go" "APIUnitsCmdOutput"]]

## Standalone Commands

Standalong commands are for the srclib power user: most people will use srclib through an editor plugin or Sourcegraph, but the following commands are useful for modifying the state of a repository's analysis data.

### `srclib config`
`srclib config` is used to detect what kinds of source units (npm/pip/Go/etc. packages) exist in a repository or directory tree.

### `srclib make`
`srclib make` is used to perform analysis on a given directory. See the [srclib make docs](make.md) for usage instructions.
