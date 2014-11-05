# Getting started

## Installing srclib

To install the **src** program, download one of the prebuilt binaries or build
it from source (see next section).

src binary downloads:

* [Linux amd64](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=linux&arch=amd64&channel=stable)
* [Linux i386](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=linux&arch=386&channel=stable)
* [Mac OS X](https://api.equinox.io/1/Applications/ap_BQxVz1iWMxmjQnbVGd85V58qz6/Updates/Asset/src.zip?os=darwin&arch=amd64&channel=stable)

After downloading the file, unzip it and place the `src` program in your
`$PATH`. Run `src --help` to verify that it's installed.

To install the standard set of language analysis toolchains
([Go](toolchains/go.md), [Ruby](toolchains/ruby.md),
[JavaScript](toolchains/javascript.md), and [Python](toolchains/python.md)), run:

```
src toolchain install-std
```

By default this installs the toolchains for Ruby, Go, JavaScript, and Python. To skip installing toolchains you don't care about, use `--skip`, as in `src toolchain install-std --skip javascript --skip go`.

If this command fails, please
[file an issue](https://github.com/sourcegraph/srclib/issues).

Now, `src toolchain list` should show the toolchains you just installed.

### Building from source

First, [install Go](http://golang.org/doc/install) (version 1.3 or newer).

Then download and install `src`:

```
go get -u -v sourcegraph.com/sourcegraph/srclib/cmd/src
```

Next, you must install the language toolchains: `src toolchain install-std`. By default this installs the toolchains for Ruby, Go, JavaScript, and Python. To skip installing toolchains you don't care about, use `--skip`, as in `src toolchain install-std --skip javascript --skip go`.

Now, `src toolchain list` should show the toolchains you just installed.

## Next steps

### Download an editor plugin

If you are interested in using the editor plugins that we have available, check
out the Editor Plugins section of the documentation. Currently,
[Emacs](plugins/emacs.md) [Sublime Text](plugins/sublimetext.md) and
[Atom](plugins/atom.md) are supported, and support for more editors is coming soon.

### Build on srclib

If you want to help build/improve editor plugins, or simply hack on srclib,
read through the docs on [Building on srclib](api/overview.md).

### Contribute to srclib

Finally, if you want to help build out the language analysis infrastructure,
make sure you're familiar with the [API](api/overview.md). Then, read closely over
the [Creating a Toolchain](toolchains/overview.md) section.
