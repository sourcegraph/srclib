# srclib toolchains

A toolchain adds support for a programming language to srclib by implementing
the following components.

* Scanner: finds all *source units* of the language (e.g., Python packages, Ruby gems, etc.) in a directory tree.
* Dependency lister: enumerates all dependencies specified by the language's source units in their raw form (e.g., as the name and version of an npm or pip package).
* Dependency resolver: resolves raw dependencies to git/hg clone URIs, subdirectories, and commit IDs if possible (e.g., `foo@0.2.1` to github.com/alice/foo commit ID abcd123).
* Grapher: performs type checking/inference and static analysis (called "graphing") on the language's source units and dumps data about all definitions and references.

## Example: Python toolchain

TODO

# Toolchain discovery

Toolchains are programs in your `$PATH` whose name matches `src-*-toolchain`. To
determine the list of available toolchains, the `src` tool simply enumerates all
such programs.

To see a list of available toolchains, run `src toolchains`.

## Command-line specifications

Toolchains (i.e., programs named `src-*-toolchain`) must adhere to the following
command-line protocol.

| Command-line args | Output                        |
| `--version`       | Human-readable version string |
| `info`            | Human-readable verbose info   |
