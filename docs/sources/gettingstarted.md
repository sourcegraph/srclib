# Getting Started

## Install srclib

Eventually, binary distributions will be available, bundled with default toolchains.
For now, srclib can be built easily from source. First, you will need to install Go - instructions are available
[here](http://golang.org/doc/install).

Once Go is installed, The `src` command-line tool can then
be installed with the following command.

```
go get -v sourcegraph.com/sourcegraph/srclib/cmd/src
```
Next, you must install the individual toolchains for each language you wish to use. See instructions at:

* [**srclib-go**](toolchains/go.md) for Go
* [**srclib-javascript**](toolchains/javascript.md) for JavaScript (Node.js)

You should now be able to use and contribute to srclib.

## Next Steps

If you're interested in srclib, here are a couple next steps that you can take.

### Download an Editor Plugin

If you are interested in using the editor plugins that we have available, check
out the Editor Plugins section of the documentation. Currently, only an [Emacs plugin](plugins/emacs.md) is available, but others
are in the works.

### Build on srclib

If you want to help build/improve editor plugins, or simply hack on srclib,
read through the docs on [Building on srclib](api/overview.md).

### Contribute to srclib

Finally, if you want to help build out the language analysis infrastructure,
make sure you're familiar with the [API](api/overview.md). Then, read closely over
the [Creating a Toolchain](creatingtoolchain/overview.md) section.
