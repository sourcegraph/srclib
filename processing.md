# Processing repositories

The previous (current) processing system suffered from several flaws:

* It was often necessary to perform repository-level tasks in the grapher (such as `GOPATH` setup, `go get`, or for Python graphing perf), but the interface only allowed per-source-unit operation.
* The state and data flow was complex, involving many steps: the repo job runner, the repo task runner, the source unit scanner, the 3 grapher interface methods (Dep, Build, and Analyze), the external grapher program, and the callbacks. Inspecting the state or data at each point was difficult.
* Language dependencies (such as Python, pip, Go, Ruby, rubygems, Node.js, npm, Java, etc.) and build dependencies (C libs, g++, etc.) were installed systemwide, which made deployment take a long time and tied us to specific versions (which had different behavior on different Ubuntus and Mac).
* We frequently ran untrusted code with no protections (when installing repository dependencies, etc.).

Sourcegraph's new processing scheme is designed to adhere to these principles:

* Each step should have explicit inputs that are easy to inspect.
* Untrusted code should run sandboxed in Docker containers.
* Language or repository dependencies should be installed sandboxed in Docker containers, not systemwide.



## Planning

Planning consists of 3 steps: scanning, configuration, and planning.


### 1. Scan repository to automatically detect source units

The planner first calls `scan.SourceUnits(dir)` to automatically detect source
units in the repository. Not all information about each source unit can be
determined at this stage; some will be filled in in the next step. (For example,
we can't determine Go import paths. Even if you point it to a directory that's
in a valid `GOPATH` on your dev machine, we can't assume that to be the case
when running on the server.)

Also, the source units detected in this step can be overridden or excluded using
a `.sourcegraph` configuration file (see the next section).

To see the auto-detected source units in a directory, use `sg-scan`:

```bash
$ godep go run cmd/sg-scan/sg-scan.go $HOME/my_go_library
&unit.GoPackage{Dir:"/home/sqs/my_go_library", ImportPath:""}
```


### 2. Configure source units

A source unit's configuration specifies the kind of thing it is (a Go package, a
RubyGem, etc.), the files that comprise it, and build settings (pre-build
scripts, build tags, interpreter versions, etc.).

The source unit scanner guesses the configuration for each source unit it
detects. Repository authors may override the guesses by using a `.sourcegraph`
manual configuration file.

The **final configuration** of a repository is the result of merging the
auto-detected and manual configuration (if any). It, along with the repository
checkout itself, must contain all of the information necessary to process the
repository. (Any special handling of specific repositories should be implemented
by emitting special configuration, not with if-statements in the analysis code.)

To see the final configuration for a repository, use `sg-config`. For example:

```bash
$ godep go run cmd/sg-config/sg-config.go $GOPATH/src/github.com/sqs/spans github.com/sqs/spans
```

```toml
[[go_package]]
  dir = "."
  import_path = "github.com/sqs/spans"

[env]
  [env.go]
    base_import_path = "github.com/sqs/spans"
```

*What does this mean?* The first part specifies that there is a single Go package
source unit in the top level of the repository directory, with an import path of
`github.com/sqs/spans`. The `[env]` section contains repository-wide
configuration; here, it specifies the Go import path of the repository, which
the Go grapher must know to set up the correct `GOPATH`.


### 3. Plan what to do

The **planner** takes the repository's final configuration and determines what
needs to be done to process the repository. Currently there are 3 kinds of
tasks:

* **VCS log and blame:** fetch the list of commits and blame each file. This is language-independent and does not require running any untrusted code. It requires full commit and file history.
* **Dependency history and resolution:** find each source unit's dependencies as of each commit. This is language-specific and requires full commit and file history.
* **Graph source units.** This is language specific and requires running untrusted code. It operates on a single commit.

As an optimization, the planner might check for cached results that allow some
of these tasks to be skipped. For example, if a previous revision's processed
output already exists in the database, the planner might only perform these
tasks on source units that changed.



### 4. Specify a base environment and actions for each source unit

For each source unit, the planner determines the environment in which
processing will occur. This entails specifying:

* the base VM image (typically the newest Ubuntu)
* commands to install language toolchains (interpreters/compilers/analyzers)
* commands to install system and library dependencies
* commands to check out and build the source unit
* commands to process the source unit for graph analysis and dependency analysis

These specifications are custom to each source unit type.
