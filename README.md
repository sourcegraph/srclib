# srcgraph

**srcgraph** is a command-line tool for automatically generating documentation
and examples for your code and updating it on
[Sourcegraph](https://sourcegraph.com).


## Install

Download one of the pre-built binaries (TODO), or install from source:

```bash
$ go get sourcegraph.com/sourcegraph/srcgraph/cmd/srcgraph
```


# Usage

To generate docs and examples for your code, go to your repository's top-level
directory and run:

```bash
srcgraph make
```

The `srcgraph make` command first determines what needs to be done (e.g., by
detecting the languages of the repository's code) and produces a Makefile that
does the work. Then it executes the Makefile.

The output is written to `/tmp/sg/github.com/USER/REPO/COMMITID` (assuming your
build dir is /tmp/sg). To display the output on
[Sourcegraph.com](https://sourcegraph.com), so that people can browse
documentation and examples online, run:

```bash
srcgraph push
```

In a few minutes, the latest documentation and examples will be visible on
[Sourcegraph.com](https://sourcegraph.com).

(You can also run `srcgraph push` without running `srcgraph make`. In that case,
all processing will occur on the Sourcegraph backend.)