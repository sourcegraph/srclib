page_title: Overview

# Overview
<div class="alert alert-danger" role="alert">Note: The API is still in flux, and may change throughout the duration of this beta.</div>

The srclib API will be used through the invocation of subcommands of the `src` executable.
Eventually, we may use a persistent `src` executable that provides a REST-based
web service.

## Commands

### `src config`
`src config` is used to detect what kinds of source units (npm/pip/Go/etc. packages) exist in a repository or directory tree.

### `src make`
`src make` is used to perform analysis on a given directory. See the [src make docs](make.md) for usage instructions.

### `src api describe`
`src api describe` will retrieve information about an identifier at a specific position in a file.
See the [src api describe docs](describe.md) for usage information and output schema.

## Starting points
First, make sure you have a high-level understanding of [srclib's data model](data-model.md).

If you understand structure of srclib and want to build on top of it, the emacs plugin source is a good place to look for reference.
View the [plugin's Lisp source](https://github.com/sourcegraph/emacs-sourcegraph-mode/blob/master/sourcegraph-mode.el).
