# srclib [![Build Status](https://travis-ci.org/sourcegraph/srclib.png?branch=master)](https://travis-ci.org/sourcegraph/srclib)

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
* **Languages:** [Go](https://github.com/sourcegraph/srclib-go),
  [Python](https://github.com/sourcegraph/srclib-python),
  [JavaScript](https://github.com/sourcegraph/srclib-javascript), and
  [Ruby](https://github.com/sourcegraph/srclib-ruby)
* **Integrations:** [Sourcegraph.com](https://sourcegraph.com) and
  [emacs-sourcegraph-mode](https://github.com/sourcegraph/emacs-sourcegraph-mode)
* **Features:** jump-to-definition, find usages, type inference, documentation
  generation, and dependency resolution

## Usage

Most people will use srclib indirectly, through editor plugins or
[Sourcegraph.com](https://sourcegraph.com). Visit
[srclib.org](http://srclib.org) for editor plugin installation instructions.

For dev tools hackers: The included `src` program invokes language-specific
analysis toolchains on repositories, produces output in standardized formats,
and exposes an API for editor integration. Language toolchains are programs that
adhere to a spec and otherwise perform entirely language-specific analysis, and
tooolchains may be easily installed and modified by users.

# Misc.

* **bash completion**: run `source contrib/completion/src-completion.bash` or
  copy that file to `/etc/bash_completion.d/srclib_src` (path may be different
  on your system)
