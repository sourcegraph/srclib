# srclib [![Build Status](https://travis-ci.org/sourcegraph/srclib.png?branch=master)](https://travis-ci.org/sourcegraph/srclib)

*Note: srclib is alpha.
[Post an issue](https://github.com/sourcegraph/srclib/issues) if you have any
questions or difficulties running and hacking on it. We'll have a full docs site
up soon.*

**srclib** is a source code analysis library. It provides standardized tools,
interfaces and data formats for generating, representing and querying
information about source code in software projects.

**Why?** Right now, most people write code in editors that don't give them as
much programming assistance as is possible. That's because creating an editor
plugin and language analyzer for your favorite language and editor combo is a
lot of work. And when you're done, your plugin only supports a single language
and editor, and maybe only half the features you wanted (such as doc lookups and
"find usages"). Because there are no standard cross-language and cross-editor
APIs and formats, it is difficult to reuse your plugin for other languages or
editors.

We call this the **M-by-N-by-O problem**: given *M* editors, *N* languages, and
*O* features, we need to write (on the order of) *M*&times;*N*&times;*O* plugins
to get good tooling in every case. That number gets large quickly, and it's why
we suffer from poor developer tools.

srclib solves this problem in 2 ways by:

* Publishing standard formats and APIs for source analyzers and editor plugins
  to use. This means that improvements in a srclib language analyzer benefit
  users in any editor, and improvements in a srclib editor plugin benefit
  everyone who uses that editor on any language.

* Providing high-quality language analyzers and editor plugins that implement
  this standard. These were open-sourced from the code that powers
  [Sourcegraph.com](https://sourcegraph.com).

Currently, srclib supports:

* **Languages:** [Go](https://sourcegraph.com/sourcegraph/srclib-go) and [JavaScript](https://sourcegraph.com/sourcegraph/srclib-javascript) (coming very soon: [Python](https://sourcegraph.com/sourcegraph/srclib-python) and [Ruby](https://sourcegraph.com/sourcegraph/srclib-ruby))

* **Integrations:** [Sourcegraph.com](https://sourcegraph.com) and
  [emacs-sourcegraph-mode](https://sourcegraph.com/sourcegraph/emacs-sourcegraph-mode)

* **Features:** jump-to-definition, find usages, type inference, documentation
  generation, and dependency resolution

Want to extend srclib to support more languages, features, or editors? During
this alpha period, we will work closely with you to help you.
[Post an issue](https://github.com/sourcegraph/srclib/issues) to let us know
what you're building to get started.


## Usage

srclib requires Go 1.2+, Git, and Mercurial.

Install and run `src`, the command-line frontend to srclib, with:

```
go get -v sourcegraph.com/sourcegraph/srclib/cmd/src
```

Next, install toolchains for the languages you want to use. See instructions at:

* [**srclib-go**](https://sourcegraph.com/sourcegraph/srclib-go) for Go
* [**srclib-javascript**](https://sourcegraph.com/sourcegraph/srclib-javascript) for JavaScript (Node.js)

Finally, install an editor plugin powered by srclib:

* [**emacs-sourcegraph-mode**](https://sourcegraph.com/sourcegraph/emacs-sourcegraph-mode) for Emacs

Don't see your language or editor of choice?
[Create or +1 an issue](https://github.com/sourcegraph/srclib/issues) to vote
for it, or start adding support for it yourself!

# Misc.

* **bash completion** for `src`: run `source contrib/completion/src-completion.bash` or
  copy that file to `/etc/bash_completion.d/srclib_src` (path may be different
  on your system)
