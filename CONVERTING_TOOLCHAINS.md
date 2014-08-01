# Converting toolchains the previous internal Sourcegraph structure

**Previously**, toolchains were Go packages that were compiled together with the
runner tool. Toolchain packages implemented a few Go interfaces:

```golang
type Scanner interface {
    // Scan returns a list of source units that exist in dir and its
    // subdirectories. Paths in the source units should be relative to dir.
    Scan(dir string, c *config.Repository) ([]unit.SourceUnit, error)
}

type Grapher interface {
    Graph(dir string, unit unit.SourceUnit, c *config.Repository) (*Output, error)
}

type Resolver interface {
    Resolve(dep *RawDependency, c *config.Repository) (*ResolvedTarget, error)
}

type Lister interface {
    List(dir string, unit unit.SourceUnit, c *config.Repository) ([]*RawDependency, error)
}
```

and registered their functionality by calling
`{unit,grapher2,scan,dep2}.Register*` functions.

**Now**, toolchains are separate programs and/or Dockerfiles from the runner.
See TOOLCHAINS.md for a description and specification.

This document explains how to convert a toolchain from the old structure to the
new structure. Once they've all been converted, this document will be adapted to
explain how to create a new language toolchain from scratch.

## 1. Create a separate repository for the toolchain

Use the scripts in the sourcegraph/devtools repository to create a new repository:

```
$PATH_TO_DEVTOOLS_REPO/scripts/github-setup -f $FLOWDOCK_TOKEN -t $GITHUB_TOKEN new-repo srclib-mylang
$PATH_TO_DEVTOOLS_REPO/scripts/init-repo-from-template srclib-mylang
```

Then copy the code over using:

```
# copy git history from the old toolchain directory tree in the main sourcegraph repo
git subtree split -P srcgraph/toolchain/mylang -b tmp-mylang
cd $PATH_TO_NEW_REPO
git pull $PATH_TO_MAIN_SOURCEGRAPH_REPO tmp-mylang

# move history in srcgraph/toolchain/mylang to the root of the new repository
$PATH_TO_DEVTOOLS_REPO/scripts/rewrite-git-history-in-subdir srcgraph/toolchain/mylang
```

(For all of the existing toolchains, repositories have already been created with
copied history.)


## 2. Set up the toolchain's build system to produce a program at `.bin/NAME`

We will eventually implement both Docker and program execution methods for the
toolchain, but let's start by just making it runnable as a program, since that
has the quickest dev cycles.

We'll also might want to port the toolchain code from Go to whatever language
they target, but let's keep them as Go for now (at least for this step).

The easiest way to make Go build a binary at `.bin/NAME` is to put the main
package in the repository root directory and create a Makefile containing:

```
.PHONY: install

install:
	@mkdir -p .bin
	go build -o .bin/srclib-mylang
```

(This Makefile is not used by srclib, but it is helpful to have during
development so you don't have to remember the `go build` command.)

This means you should move all non-main-package code to a subpackage, such as
`sourcegraph.com/sourcegraph/srclib-mylang/mylang`.

## 3. Create a Srclibtoolchain to describe the toolchain

In the toolchain repository, make a Srclibtoolchain file containing:

```
{
  "Tools": [
    {
      "Subcmd": "scan",
      "Op": "scan",
      "SourceUnitTypes": [
        "mylangpkg"
      ]
    },
    {
      "Subcmd": "graph",
      "Op": "graph",
      "SourceUnitTypes": [
        "mylangpkg"
      ]
    },
    {
      "Subcmd": "depresolve",
      "Op": "depresolve",
      "SourceUnitTypes": [
        "mylangpkg"
      ]
    }
  ]
}
```

where `mylangpkg` is the existing source unit type of the toolchain you're
converting (`PipPackage`, etc.).

This file tells the `src` tool about your toolchain.

## 4. Add the toolchain to your SRCLIBPATH

The SRCLIBPATH is where `src` discovers toolchains. It defaults to `~/.srclib`.

Assuming you've checked out the toolchain repository to `$HOME/src/sourcegraph.com/sourcegraph/srclib-mylang`, run the following to add that toolchain to your SRCLIBPATH:

```
ln -s $HOME/src/sourcegraph.com/sourcegraph/srclib-mylang $HOME/.srclib/sourcegraph.com/sourcegraph/srclib-mylang
```

Now `src toolchain list` should display your toolchain, and `src toolchain
list-tools` should display the tools you defined in the Srclibtoolchain.

## 5. Implement the various tools (subcommands) in the toolchain program

We need the main package to produce a binary that responds to a specific set of
subcommands (`scan`, `depresolve`, `graph`) and flags (see TOOLCHAINS.md for
details). The srclib-go repository's
[cli.go](https://sourcegraph.com/sourcegraph/srclib-go/blob/master/cli.go) is a good
template to use to create a program that adheres to this spec.

The quickest way to test whether your program works is to just have it emit a
constant JSON string in response to each subcommand.

## 6. Try your toolchain on a real repository

Take one of the sample repositories in sgtest for the language whose toolchain
you're converting. Run `src do-all` from within that sample repository. Does it
run your toolchain? Are there any errors?

## 7. Create a Dockerfile that wraps your toolchain

In the root directory of the srclib-mylang repository, create a Dockerfile that
installs the necessary dependencies to run your toolchain and sets the toolchain
as the Docker container's entrypoint. See the srclib-go toolchain's
[Dockerfile](https://sourcegraph.com/sourcegraph/srclib-go/blob/master/Dockerfile)
for an example.

When `src` runs your Docker container, it will always mount the project's source
code at `/src`.

If you've created your Dockerfile correctly, then the following should build the image:

```
src toolchain build sourcegraph.com/sourcegraph/srclib-mylang
```

Then running `src toolchain list` should show that this toolchain is Docker-capable.

Now run `src make -m docker` in that same sgtest sample repository. The `-m
docker` tells it to use *only* the Docker execution method (i.e., don't run it
as a local program). If you've done everything correctly so far, you should get
no errors and similar output as when you ran `src make` by itself (which runs
your toolchain as a local program).

## 8. Implement the actual tool (subcommand) functionality

If you just made the tools print a constant JSON string, now go and implement them for real.

Because the Dockerfile simply wraps your program, when you add the functionality
to the program, your Dockerized toolchain will also have it. However, be sure to
rebuild the Docker image when the program changes, by running `src toolchain
build sourcegraph.com/sourcegraph/srclib-mylang`.

## 9. Create a test case

The `src test` helps you test your toolchain. It automatically detects test
cases in toolchain repositories that are located in the `test/case` dir.

To add a test case from an existing repository, run:

```
git submodule add git@github.com:sgtest/mylang-sample.git testdata/case/mylang-sample
```

(Substitute `mylang-sample` for the name of an existing sample repository.)

Then run `src test --gen` to create the expected output in the
`testdata/expected/mylang-sample` directory. Fix any errors that prevent this
output from being written, of course, and manually check the output to ensure it
is the same output as was generated by the previous-generation toolchain.

Once `src test --gen` successfully writes expected output, let's try running the
test with `src test`. There should be no diffs and it should say `mylang-sample
PASS`. If not, check the diffs and fix accordingly.

Commit the expected output and the submodule addition to your repository.

## 10. You're done!

So, you have a toolchain that runs as both a program and as a Docker container,
and it passes on all of the existing sample repositories. Nice job! You're done.

# Caveats

There are some places where the `srclib-go` toolchain is hardcoded. Until this
hardcoding is removed, be sure to change all references to `srclib-go` to your
new toolchain in the `srclib` repository, and rebuild the `src` program.
