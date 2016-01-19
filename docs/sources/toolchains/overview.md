# Language toolchains

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
toolchain. To see all available toolchains, run `srclib info toolchains`.

## Tool discovery

A toolchain's tools are described in its Srclibtoolchain file. To see all
available tools (provided by all available toolchains), run `srclib info tools`.


# Running tools

Tools are normal programs on your system. They rely on locally
installed compiler/interpreter and dependency versions.

An installed tool is an executable program located at "TOOLCHAIN/.bin/NAME",
where TOOLCHAIN is the toolchain path and NAME is the last component in the
toolchain path. For example, the installed tool for "github.com/foo/bar"
would be at "SRCLIBPATH/github.com/foo/bar/.bin/bar".

Tools and toolchains are not typically invoked directly by users; the
`srclib` program invokes them as part of higher-level
commands. However, it is possible to invoke them directly. To run a
tool, run:

```
srclib tool TOOLCHAIN TOOL [ARG...]
```

For example:

```
srclib tool github.com/alice/srclib-python scan ~/my-python-project
```


# Toolchain & tool specifications

Toolchains and their tools must conform to the protocol described
below. This includes four subcommands, listed below:

## info
This command should display a human-readable info describing
the version, author, etc. (free-form)

## scan (scanners)

Tools that perform the `scan` operation are called **scanners**. They scan a
directory tree and produce a JSON array of source units (in Go,
`[]*unit.SourceUnit`) they encounter.

**Arguments:** none; scanners scan the tree rooted at the current directory (typically the root directory of a repository)

**Stdin:** JSON object representation of repository config (typically `{}`)

**Stdout:** `[]*unit.SourceUnit`. For a more detailed description, [read the scanner output spec](scanner-output.md).

See the `scan.Scan` function for an implementation of the calling side of this
protocol.



## depresolve (dependency resolvers)

Tools that perform the `dep` operation are called **dependency resolvers**. They
resolve "raw" dependencies, such as the name and version of a dependency
package, into a full specification of the dependency's target.

**Arguments:** none

**Stdin:** JSON object representation of a source unit (`*unit.SourceUnit`)

**Options:** none

**Stdout:** `[]*dep.Resolution` JSON array with each item corresponding to the
same-index entry in the source unit's `Dependencies` field. For a more
detailed description, [read the dependency resolution output sepc](dependency-resolution-output.md).

## graph  (graphers)

Tools that perform the `graph` operation are called **graphers**. Depending on
the programming language of the source code they analyze, they perform a
combination of parsing, static analysis, semantic analysis, and type inference.
Graphers perform these operations on a source unit and have read access to all
of the source unit's files.

**Arguments:** none

**Stdin:** JSON object representation of a source unit (`*unit.SourceUnit`)

**Options:** none

**Stdout:** JSON graph output (`grapher.Output`). field. For a more
detailed description, [read the grapher output spec](grapher-output.md).

<!---
TODO(sqs): Can we provide the output of `dep` to the `graph` tool? Usually
graphers have to resolve all of the same deps that `dep` would have to. But
we're already providing a full JSON object on stdin, so making it an array or
sending another object would slightly complicate things.
--->

# Available Toolchains

<!--- Stolen from overview.md. --->

<ul>
  <li><a href="go.md">Go</a></li>
  <li><a href="java.md">Java</a></li>
  <li><a href="python.md">Python</a></li>
  <li><a href="javascript.md">JavaScript</a></li>
  <li><a href="haskell.md">Haskell</a></li>
  <li><a href="ruby.md">Ruby (WIP)</a></li>
  <li><a href="php.md">PHP (WIP)</a></li>

</ul>
