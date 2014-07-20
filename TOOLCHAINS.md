# srclib toolchains

A toolchain is a git repository that adds functionality to srclib by
implementing any of the following tools.

* Scanner: runs before all other tools and finds all *source units* of the
  language (e.g., Python packages, Ruby gems, etc.) in a directory tree.
  Scanners also determine which other tools to call on each source unit.
* Dependency lister: enumerates all dependencies specified by the language's
  source units in their raw form (e.g., as the name and version of an npm or pip
  package).
* Dependency resolver: resolves raw dependencies to git/hg clone URIs,
  subdirectories, and commit IDs if possible (e.g., `foo@0.2.1` to
  github.com/alice/foo commit ID abcd123).
* Grapher: performs type checking/inference and static analysis (called
  "graphing") on the language's source units and dumps data about all
  definitions and references.

A toolchain is distributed as a git repository and is identified by its "clone
URI", such as "github.com/alice/srclib-python".

Toolchain repositories may contain any number of tools. For example, a
repository could implement only a Python scanner, and the scanner would specify
that the Python packages it finds should be graphed by a tool in a different
repository.

Each tool in a toolchain is identified by the toolchain path plus the tool's
path within that repository, such as "github.com/alice/srclib-python/scan".

Repository authors can choose which toolchains and tools to use in their
project's `.sourcegraph` file. If none are specified, the defaults apply.

TODO(sqs): Should we call the config file `.sourcegraph` or something else?


# Tool discovery

The SRCLIBPATH environment variable lists places to look for srclib tools. The
value is a colon-separated string of paths. If it is empty, `~/.srclib` is used.

If DIR is a directory listed in SRCLIBPATH, the directory
"DIR/github.com/foo/bar" defines a tool named "github.com/foo/bar".

TODO(sqs): Add a way to enumerate all available tools. Add a notion of a
toolchain that groups/describes the tools within it.


# Running tools

There are 2 ways to run srclib tools:

1. Inside a Docker container: to produce analysis independent of your local
   configuration and versions. (Used when other people or services will reuse
   the analysis results, such as on [Sourcegraph](https://sourcegraph.com).)
1. Directly, as a normal program on your system: to produce analysis that relies
   on locally installed compiler/interpreter and dependency versions. (Used when
   you'll consume the analysis results locally, such as during editing of local
   code.)
   
Tools may support either or both of these execution methods.

## Docker-containerized tools

A tool whose directory (under SRCLIBPATH) contains a Dockerfile can be run
within a Docker container built using that Dockerfile.

## Directly runnable tools

A tool whose directory (under SRCLIBPATH) contains a Dockerfile with an
`ENTRYPOINT` instruction can be run directly.

When running directly, it's assumed that your local machine already has been
configured such that the Dockerfile's `ENTRYPOINT` will also invoke the correct
program locally. If this is not the case, you may override the entrypoint with
the `X-SRCLIB-DIRECT-ENTRYPOINT` directive. (Docker ignores unrecognized
directives such as this one.) It's recommended to create your Dockerfiles so
that their `ENTRYPOINT` is also the command that you'd run locally to invoke the
tool.

TODO(sqs): Implement `X-SRCLIB-DIRECT-ENTRYPOINT`.

Note: Using the Dockerfile to determine how to run the tool directly is a
short-term hack. We'll probably come up with a better solution.

## Example

See [srclib-sample](https://github.com/sourcegraph/srclib-sample).


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

## Scanner protocol

Scanners must implement the following commands:

| Command                      | Description                                    | JSON output (Go type) |
| `scan DIR`                   | Discover source units in DIR (and its subdirs) | scan.Output           |

