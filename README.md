# srclib [![Build Status](https://travis-ci.org/sourcegraph/srclib.png?branch=master)](https://travis-ci.org/sourcegraph/srclib)

*Note: srclib is alpha.
[Post an issue](https://github.com/sourcegraph/srclib/issues) if you have any
questions or difficulties running and hacking on it.*

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

srclib requires Go 1.2+, Git, and Mercurial. Install and run srclib with:

```
# download and install 'src', the command for running srclib
go get -v sourcegraph.com/sourcegraph/srclib/cmd/src

# install toolchain for JavaScript to ~/.srclib
mkdir -p ~/.srclib/sourcegraph.com/sourcegraph
cd ~/.srclib/sourcegraph.com/sourcegraph
git clone https://github.com/sourcegraph/srclib-javascript
cd srclib-javascript && npm install && cd node_modules/jsg && npm install

# check that the toolchain is installed
src toolchain list
# should show 'sourcegraph.com/sourcegraph/srclib-javascript'

# try it on a sample JavaScript repository
cd /tmp
git clone https://github.com/sgtest/javascript-nodejs-xrefs-0.git
cd javascript-nodejs-xrefs-0
src do-all
# it writes analysis output to .srclib-cache/...

# query the analysis output:
src api describe --file $PWD/lib/http.js --start-byte 4
src api describe --file $PWD/lib/http.js --start-byte 100
# you should see JSON describing what's defined at that position in the file
```

OK, now srclib is installed. There are 2 ways to use it:

### As an editor plugin backend (most common)

srclib powers high-quality language support in your favorite editor. Currently
the only available plugin is
[emacs-sourcegraph-mode](https://sourcegraph.com/sourcegraph/emacs-sourcegraph-mode),
but people are building more right now.

To use an Sourcegraph editor plugin powered by srclib, follow the instructions
in the editor plugin's README.

### As a source analysis tool, or extending srclib itself to support more languages and editors (for dev tools hackers)

The included `src` program invokes language-specific analysis toolchains on
repositories, produces output in standardized formats, and exposes an API for
editor integration. Language toolchains are programs that adhere to a spec and
otherwise perform entirely language-specific analysis, and tooolchains may be
easily installed and modified by users.

For toolchains, we have a work-in-progress spec describing how to build them. (TODO add link)

For editor plugins, run `src api describe --help` to see the command API, and
check out
[emacs-sourcegraph-mode](https://sourcegraph.com/sourcegraph/emacs-sourcegraph-mode)
for a reference implementation.


# Misc.

* **bash completion** for `src`: run `source contrib/completion/src-completion.bash` or
  copy that file to `/etc/bash_completion.d/srclib_src` (path may be different
  on your system)
