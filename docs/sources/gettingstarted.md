# Getting Started

## Installing srclib

To install the **src** program, download one of the prebuilt binaries or build
it from source (see next section).

src binary downloads:

* [Linux amd64](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=darwin&arch=amd64&channel=stable)
* [Linux i386](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=darwin&arch=386&channel=stable)
* [Mac OS X](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=darwin&arch=amd64&channel=stable)

After downloading the file, unzip it and place the `src` program in your
`$PATH`. Run `src --help` to verify that it's installed.

### Building from source

First, [install Go](http://golang.org/doc/install) (version 1.3 or newer).

Then download and install `src`:

```
go get -u -v sourcegraph.com/sourcegraph/srclib/cmd/src
```

Next, you must install the toolchain for each language you wish to use. See
instructions at:

* [**srclib-go**](toolchains/go.md) for Go
* [**srclib-javascript**](toolchains/javascript.md) for JavaScript (Node.js)
* [**srclib-ruby**](toolchains/ruby.md) for Ruby

To check which toolchains are installed, run `src toolchain list`.

If you see your language of choice in that list, you should now be able to use
and contribute to srclib.

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
