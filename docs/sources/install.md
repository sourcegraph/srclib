# Getting started


## Install a prebuilt binary

To install the **srclib** program, download one of the prebuilt binaries or build
it from source (see next section).

srclib binary downloads:

* [Linux amd64](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=linux&arch=amd64&channel=stable)
* [Linux i386](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=linux&arch=386&channel=stable)
* [Mac OS X](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=darwin&arch=amd64&channel=stable)

After downloading the file, unzip it and place the `srclib` program in your
`$PATH`. Run `srclib --help` to verify that it's installed.


## Building from source

First, install [install Go](http://golang.org/doc/install) (version 1.5 or newer).

Then:

```
go get -u -v sourcegraph.com/sourcegraph/srclib/cmd/srclib
```

<br>

##Language Toolchains

To install the language analysis toolchains for
([Go](toolchains/go.md), [Ruby](toolchains/ruby.md),
[JavaScript](toolchains/javascript.md), and [Python](toolchains/python.md)), run:

```
srclib toolchain install go ruby javascript python
```

If this command fails, please
[file an issue](https://github.com/sourcegraph/srclib/issues) or skip
one of the languages if you don't need it.

`srclib toolchain list` helps to verify the currently installed language toolchains like the following example

```
$ srclib toolchain list
sourcegraph.com/sourcegraph/srclib-python
sourcegraph.com/sourcegraph/srclib-go
```

<br>

##Testing srclib

In order to test srclib we can use it to analyze the already fetched source code for the Go toolchain `srclib-go`.

First, you need to initialize the git submodules in the root directory of srclib-go
```
cd src/sourcegraph.com/sourcegraph/srclib-go
git submodule update --init
```

Now you can test the srclib-go toolchain with:

```
$ srclib do-all
$ srclib store import
```

You should have a .srclib-cache directory inside srclib-go that has all of the
build data for the repository. You should also have a .srclib-store directory
corresponding to the analysis information.

```
srclib store defs
```

should show you the definitions.

<br>
