# srclib tools

A tool is a program that implements some of the following functionality (as
individual subcommands) for analyzing software projects and source code:

* Scanning: runs before all other tools and finds all *source units* of the
  language (e.g., Python packages, Ruby gems, etc.) in a directory tree.
  Scanners also determine which other tools to call on each source unit.
* Dependency listing: enumerates all dependencies specified by the language's
  source units in their raw form (e.g., as the name and version of an npm or pip
  package).
* Dependency resolution: resolves raw dependencies to git/hg clone URIs,
  subdirectories, and commit IDs if possible (e.g., `foo@0.2.1` to
  github.com/alice/foo commit ID abcd123).
* Graphing: performs type checking/inference and static analysis (called
  "graphing") on the language's source units and dumps data about all
  definitions and references.

Tools may contain any number of subcommands. For example, a tool could implement
only a Go scanner, and the scanner would specify that the Go packages it finds
should be graphed by a tool in a different repository. Or a single tool could
implement the entire set of Go source analysis operations.

Srclib ships with a default set of tools for some popular programming languages.
Repository authors and srclib users may install third-party tools to add
features or override the default tools.

A tool is defined by a directory that contains a file named Srclibtool, which
describes the tool and its capabilities. A tool is identified by its
repository's clone URI (e.g., "github.com/alice/srclib-go") joined with the
tool's path within that repository, such as
"github.com/alice/srclib-python/scan".

Repository authors can choose which tools to use in their project's
`.sourcegraph` file. If none are specified, the defaults apply.

TODO(sqs): Should we call the config file `.sourcegraph` or something else?


# Tool discovery

The SRCLIBPATH environment variable lists places to look for srclib tools. The
value is a colon-separated string of paths. If it is empty, `~/.srclib` is used.

If DIR is a directory listed in SRCLIBPATH, the directory
"DIR/github.com/foo/bar" defines a tool named "github.com/foo/bar".

Tool directories may contain a Srclibtool file describing and configuring the
tool. Tools with a Srclibtool file will appear in the list printed by `src
tools`. However, a Srclibtool file is not required to run the tool; it may still
be specified manually.


# Running tools

There are 2 modes of execution for srclib tools:

1. As a normal **installed program** on your system: to produce analysis
   that relies on locally installed compiler/interpreter and dependency
   versions. (Used when you'll consume the analysis results locally, such as
   during editing of local code.)
   
   A directly runnable tool is any program in your `PATH` named `src-tool-*`.
   These programs must already be installed in your system.
1. Inside a **Docker container**: to produce analysis independent of your local
   configuration and versions. (Used when other people or services will reuse
   the analysis results, such as on [Sourcegraph](https://sourcegraph.com).)
   
   A Docker-containerized tool is a directory (under SRCLIBPATH) that contains a
   Dockerfile. There is no installation necessary for these tools; the `src` tool
   knows how to build and run their Docker container.
   
Tools may support either or both of these execution modes.

To run a tool, run:

```
src tool TOOL [ARG...]
```

The `TOOL` argument can be either the
last part of an installed program's name (e.g., `foo` in `src-tool-foo`), or the
name of a Dockerized tool in your SRCLIBPATH (e.g.,
`github.com/alice/srclib-python`).

For example:

```
# To run a tool directly named src-tool-python that's installed in your PATH:
src tool python

# To run a tool (inside a Docker container) whose repository
# github.com/alice/srclib-python/scan is in your SRCLIBPATH at
# ~/.srclib/github.com/alice/srclib-python:
src tool github.com/alice/srclib-python
```


# Tool specifications

The current list of tool types is:

* scanner
* dependency lister
* dependency resolver
* grapher

Each type of tool implements a protocol: a defined set of commands and input
arguments, and a defined output format. All tools must also implement a set of
common commands.

Commands and arguments are passed as command-line arguments to the tool, and
output is written to stdout.

The tool protocol is the same whether the tool is run directly or inside a
Docker container (assuming it supports both methods).

## Common protocol

All tools must implement the following commands:

| Command           | Description  | Output                          |
| `info`            | Show info    | Human-readable info (free-form) |
| `caps`            | 

## Scanner protocol

Scanners must implement the following commands:

| Command                      | Description                                    | JSON output (Go type) |
| `scan DIR`                   | Discover source units in DIR (and its subdirs) | scan.Output           |
