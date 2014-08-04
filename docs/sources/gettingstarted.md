# What is srclib?

The srclib project aims to provide the following functionality in a language
agnostic fashion:

1. Jump to definition support, both within local code bases, as well as across
   online repositories
1. Given a definition, find examples of its use across open source code
1. Search and find documentation for modules and functions
1. Expose an API that makes it easy to query the results of static analysis
1. Be architected in a way that makes it painless to integrate new languages

# Installing Srclib

First, you will need to install Go - instructions are available
[here](http://golang.org/doc/install).

Once Go is installed, The `src` command-line tool can then
be installed with the following command.

```
go get -v sourcegraph.com/sourcegraph/srclib/cmd/src
```
Now, you must install the individual toolchains for each language you wish to use.
Next, install toolchains for the languages you want to use. See instructions at:

* [**srclib-go**](https://sourcegraph.com/sourcegraph/srclib-go) for Go
* [**srclib-javascript**](https://sourcegraph.com/sourcegraph/srclib-javascript) for JavaScript (Node.js)

To understand how to use `src`, read through the docs, starting with the
`[overview](../src/overview.md)`.

# Next Steps

## Download an Editor Plugin

If you are interested in using the editor plugins that we have available, check
out the `[Editors](installation/editor-plugins.md)` page, and download the plugin
for your favorite editor.

## Add Support For Your Favorite Editor

If you want to help build/improve editor plugins, or simply hack on srclib,
first read through the docs on the `[src](src/overview.md)` tool. Then, check out
the `[API](api/overview.md)`.

## Help Add More Languages

Finally, if you want to help build out the language analysis infrastructure,
make sure you're familiar with the `[src](src/overview.md)` executable. Then, you
should read closely over our `[Language Analysis](language-analysis/overview.md)`
section.
