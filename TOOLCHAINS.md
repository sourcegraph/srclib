# srclib toolchains & tools

A **toolchain** is a program that implements functionality for analyzing
projects and source code, according to the specifications defined in this
document.

A **tool** is a subcommand of a toolchain that executes one particular action
(e.g., runs one type of analysis on source code). Tools accept command-line
arguments and produce output on stdout. Common operations implemented by tools
include:

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

Toolchains may contain any number of tool subcommands. For example, a toolchain
could implement only a Go scanner, and the scanner would specify that the Go
packages it finds should be graphed by a tool in a different toolchain. Or a
single toolchain could implement the entire set of Go source analysis
operations.

srclib ships with a default set of toolchains for some popular programming languages.
Repository authors and srclib users may install third-party toolchains to add
features or override the default toolchains.

A **toolchain path** is its repository's clone URI joined with the toolchain's
path within that repository. For example, a toolchain defined in the root
directory of the repository "github.com/alice/srclib-python" would have the
toolchain path "github.com/alice/srclib-python".

A tool is identified by its toolchain's path and the name of the operation it
performs. For example, "github.com/alice/srclib-python scan".

Repository authors can choose which toolchains and tools to use in their
project's Srcfile. If none are specified, the defaults apply.


# Toolchain discovery

The **SRCLIBPATH environment variable** lists places to look for srclib toolchains.
The value is a colon-separated string of paths. If it is empty, `$HOME/.srclib`
is used.

If DIR is a directory listed in SRCLIBPATH, the directory
"DIR/github.com/foo/bar" defines a toolchain named "github.com/foo/bar".

Toolchain directories must contain a Srclibtoolchain file describing and configuring the
toolchain. To see all available toolchains, run `src info toolchains`.

## Tool discovery

A toolchain's tools are described in its Srclibtoolchain file. To see all
available tools (provided by all available toolchains), run `src info tools`.


# Running tools

There are 2 modes of execution for srclib tools:

1. As a normal **installed program** on your system: to produce analysis
   that relies on locally installed compiler/interpreter and dependency
   versions. (Used when you'll consume the analysis results locally, such as
   during editing of local code.)
   
   An installed tool is an executable program located at "TOOLCHAIN/.bin/NAME",
   where TOOLCHAIN is the toolchain path and NAME is the last component in the
   toolchain path. For example, the installed tool for "github.com/foo/bar"
   would be at "SRCLIBPATH/github.com/foo/bar/.bin/bar".

1. Inside a **Docker container**: to produce analysis independent of your local
   configuration and versions. (Used when other people or services will reuse
   the analysis results, such as on [Sourcegraph](https://sourcegraph.com).)
   
   A Docker-containerized tool is a directory (under SRCLIBPATH) that contains a
   Dockerfile. There is no installation necessary for these tools; the `src`
   program knows how to build and run their Docker container.
   
Tools may support either or both of these execution modes.

Toolchains are not typically invoked directly by users; the `src` program invokes
them as part of higher-level commands. However, it is possible to invoke them
directly. To run a tool, run:

```
src tool TOOLCHAIN TOOL [ARG...]
```

For example:

```
src tool github.com/alice/srclib-python scan ~/my-python-project
```


# Toolchain & tool specifications

Toolchains and their tools must conform to the protocol described below. The
protocol is the same whether the tool is run directly or inside a Docker
container.

## Toolchain protocol

All toolchains must implement the following subcommands:

| Command           | Description  | Output                                                               |
| ----------------- | ------------ | -------------------------------------------------------------------- |
| `info`            | Show info    | Human-readable info describing the version, author, etc. (free-form) |

In addition to the `info` subcommand, each of a toolchain's tools are exposed as a subcommand.


## Scan tool

A toolchain that provides scanning functionality must implement the following subcommand:

| Command                      | Description                                    | JSON output (Go type) |
| ---------------------------- | ---------------------------------------------- | --------------------- |
| `scan DIR`                   | Discover source units in DIR (and its subdirs) | scan.Output           |
