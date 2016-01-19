# Getting started

This page is to help you get started with srclib. In order to work with srclib we need the following components:
* dependencies
* installing `srclib`
* getting language toolchains
* editor plugin (optional)

<br>


## Dependencies

Srclib depends on the following components:

* Go programming language
* Mercurial (depends on Python 2.7)


### Go 1.4+

First, you need to [install Go](http://golang.org/doc/install) (version 1.4 or newer).


### Mercurial
Mercurial is available for most platforms. Here's a few Linux examples..

Ubuntu or Debian Linux:
```
sudo apt-get update && sudo apt-get -y install mercurial
```

Centos, Fedora or Red Hat Linux:
```
sudo yum update && sudo yum install mercurial
```

<br>

## Srclib installation

To install the **srclib** program, download one of the prebuilt binaries or build
it from source (see next section).

srclib binary downloads:

* [Linux amd64](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=linux&arch=amd64&channel=stable)
* [Linux i386](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=linux&arch=386&channel=stable)
* [Mac OS X](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/srclib.zip?os=darwin&arch=amd64&channel=stable)

After downloading the file, unzip it and place the `srclib` program in your
`$PATH`. Run `srclib --help` to verify that it's installed.


### Building from source

Download and install `srclib`:

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
[file an issue](https://github.com/sourcegraph/srclib/issues).

`srclib toolchain list` helps to verify the currently installed language toolchains like the following example

```
$ srclib toolchain list
sourcegraph.com/sourcegraph/srclib-python
sourcegraph.com/sourcegraph/srclib-go
```

<br>

##Testing Srclib

In order to test srclib we can use  it to analyze  the already fetched source code for the Go toolchain `srclib-go`.

First, you need to initialize the git submodules in the root directory of srclib-go
```
cd src/sourcegraph.com/sourcegraph/srclib-go
git submodule update --init
```

now you can test the srclib-go toolchain with
```
srclib config && srclib make
```

You should have a .srclib-cache directory inside srclib-go that has all of the build data for the repository.

<br>

## Next steps

### Build on top of srclib

If you want to build or improve srclib editor plugins, read
[Building on srclib](api/overview.md).


### Contribute to srclib

If you want to help build the language analysis infrastructure, make
sure you're familiar with the [API](api/overview.md). Then, read the
[Creating a Toolchain](toolchains/overview.md) section.

<br>
