# src

**src** is a command-line tool for automatically generating documentation
and examples for your code and updating it on
[Sourcegraph](https://sourcegraph.com).


## Install

Download one of the pre-built binaries (TODO), or install from source:

```bash
$ go get github.com/sourcegraph/srclib/cmd/src
```


# Usage

To generate docs and examples for your code, go to your repository's top-level
directory and run:

```bash
src make
```

The `src make` command first determines what needs to be done (e.g., by
detecting the languages of the repository's code) and produces a Makefile that
does the work. Then it executes the Makefile.

The output is written to `/tmp/sg/github.com/USER/REPO/COMMITID` (assuming your
build dir is /tmp/sg). To display the output on
[Sourcegraph.com](https://sourcegraph.com), so that people can browse
documentation and examples online, run:

```bash
src push
```

In a few minutes, the latest documentation and examples will be visible on
[Sourcegraph.com](https://sourcegraph.com).

(You can also run `src push` without running `src make`. In that case,
all processing will occur on the Sourcegraph backend.)
