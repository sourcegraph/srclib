# What is srclib?

The srclib project aims to provide the following functionality in a language
agnostic fashion:

1. Jump to definition support, both within local code bases, as well as across
   online repositories
1. Given a definition, find examples of its use across open source code
1. Search and find documentation for modules and functions
1. Expose an API that makes it easy to query the results of static analysis
1. Be architected in a way that makes it painless to integrate new languages

# Installing srclib

First, you will need to install Go - instructions are available
[here](http://golang.org/doc/install).

Once Go is installed, The `src` command-line tool can then
be installed with the following command.

```
go get -v sourcegraph.com/sourcegraph/srclib/cmd/src
```
Next, you must install the individual toolchains for each language you wish to use. See instructions at:

* [**srclib-go**](toolchains/go.md) for Go
* [**srclib-javascript**](toolchains/javascript.md) for JavaScript (Node.js)

To understand how to use `src`, read through the info for Library Authors.

# Next Steps

## Download an Editor Plugin

If you are interested in using the editor plugins that we have available, check
out the Editor Plugins section of the documentation, and download the plugin
for your favorite editor.

## Add Support For Your Favorite Editor

If you want to help build/improve editor plugins, or simply hack on srclib,
read through the docs on [Creating a Plugin](plugins/creatingaplugin.md).

## Help Add More Languages

Finally, if you want to help build out the language analysis infrastructure,
make sure you're familiar with the [API](api/overview.md). Then, read closely over
the [Creating a Toolchain](creatingtoolchain/overview.md) section.
