no_toc: true

<h1><img alt="srclib symbol" src="../images/srclib_symbol.svg"/> <strong>srclib</strong> is a hackable, multi-language code analysis library for building better code tools.</h1>

<div class="row">
  <div class="col-sm-7">

    <p>
      srclib makes developer tools like editor plugins and code search
      better and easier to build. It supports <strong>jump to
        definition</strong>, <strong>find usages</strong>, <strong>type inference</strong>, and <strong>documentation
        generation</strong>.
    </p>

    <p>
      srclib consists of
      <a href="toolchains/overview.md"><strong>language analysis toolchains</a></strong></a>
(currently for Go, Python, JavaScript, and Ruby) with a
<a href="api/overview.md"><strong>common output format</strong></a>.
</p>

<p>
  srclib originated inside
  <a href="https://sourcegraph.com" target="_blank">&#x2731; Sourcegraph</a>, where it powers
  intelligent code search over hundreds of thousands of projects.
</p>

<!-- TODO: insert newsletter form (newsletter2.html) -->

</div>

<div class="col-sm-5">

  <!-- TODO: style buttons -->
  <ul class="action-buttons list-unstyled">
    <li><a class="btn btn-sm btn-primary" href="/install"><i class="fa fa-download"></i> Install srclib</a></li>
    <li>
      <div class="two-columns">
        <div>
          <label>Editor plugins:</label>
          <ul class="list-unstyled">
            <li><a href="plugins/emacs.md">Emacs</a></li>
            <li><a href="plugins/sublimetext.md">Sublime</a></li>
            <li><a href="plugins/atom.md">Atom</a></li>
            <li><a href="plugins/vim.md">Vim (WIP)</a></li>
            <li><a href="plugins/creatingaplugin.md" class="contribute">Create a plugin</a></li>
            <li>&nbsp;</li>
          </ul>
        </div><!--
                --><div>
          <label>Languages:</label>
          <ul class="list-unstyled">
            <li><a href="toolchains/go.md">Go</a></li>
            <li><a href="toolchains/java.md">Java</a></li>
            <li><a href="toolchains/python.md">Python</a></li>
            <li><a href="toolchains/javascript.md">JavaScript</a></li>
            <li><a href="toolchains/haskell.md">Haskell</a></li>
            <li><a href="toolchains/ruby.md">Ruby (WIP)</a></li>
            <li><a href="toolchains/php.md">PHP (WIP)</a></li>
          </ul>
        </div>
      </div><!-- <div class="two-columns"> -->
    </li>
    <li>
      <label>View code on:</label>
      <a class="btn btn-sm btn-default" target="_blank" href="https://sourcegraph.com/sourcegraph/srclib">&#x2731; Sourcegraph</button></a><!--
                                                                                                                                             --><a class="btn btn-sm btn-default" target="_blank" href="https://github.com/sourcegraph/srclib"><i class="fa fa-github"></i> GitHub</a>
</li>
</ul><!-- <ul class="action-buttons list-unstyled"> -->
</div>
</div>


<br>
## Why srclib

Srclib is designed for the purpose of making software development tools more independent from the languages they support, and to enable standard functionalities in all these tools like jump to definition, find usages, type inference, and documentation generation.

The why of srclib is explained in more detail on the [homepage](https://srclib.org/). On this page you will find an explanation of what srclib consists of, how it is used and links to more specific information.



<br>
## Components of srclib

The following components make up the core functionality of srclib:

* The `srclib` command-line analysis tool
* The `srclib api` for interacting with external applications like e.g. editor plugins
* language-specific toolchains
* common data exchange format

<br>
You can enjoy and use the functionality of srclib with

* editor plugins (jump to definition, lookup definitions on sourcegraph.com)
* a web service like sourcegraph.com
* your own extensions to srclib


<br>
## How srclib works

Srclib exposes a command-line API that you can use to analyze source code in the supported languages. The results of the analysis are stored in a well defined, language-independent format (JSON).

By using language-specific toolchains in combination with the `srclib` command, it can analyze code regardless of the language, and due to its common JSON format interaction with other tools is made much easier.


<br>
####running the `srclib make` command

When running `srclib make` on the command-line or when making API calls from extensions, in essence the following tasks are performed by the `srclib` binary:


* **Scanning** runs before all other tools and finds all source units of the language (e.g., Python packages, Ruby gems, etc.) in a directory tree. Scanners also determine which other tools to call on each source unit.


* **Dependency resolution** resolves raw dependencies to git/hg clone URIs, subdirectories, and commit IDs if possible (e.g., foo@0.2.1 to github.com/alice/foo commit ID abcd123).

* **Graphing** performs type checking/inference and static analysis (called “graphing”) on the language’s source units and dumps data about all definitions and references.

For more detailed information on this process have a look at the `srclib make` [documentation](api/make.md) .

<br>
####Using the srclib API

The srclib API has been developed to allow the components of srclib to be called from extensions like editor plugins, and to enable a standard set of features.

The API interacts with a language toolchain to support one or more of the following features:
* Jump to definition
* Show examples from sourcegraph.com

The API commands are described in more detail on the [API overview](api/overview.md) page.


<br>
#### Data exchange and format

All `srclib` tools and language toolchains interact by reading on standard input and writing to standard output.

Results of the various analysis tasks are stored in the path `$GOPATH/.srclib/` in folders with a corresponding name to the API calls. The data model and the JSON exchange format is described in more detail in the [API data model](api/overview.md).



<br>
## Getting started

**Download and install**

Check out the [installation guide](install.md) to get started with the installation of srclib.

<br>
**Get an editor plugin**

There are srclib plugins for many editors. You can scroll to the top of this page and click on one of the links.

<div class="row">
  <div class="col-md-12">
    <p>
    <a href="/plugins/emacs"><button class="btn btn-primary"><img style="height: 1em;" src="/images/editors/emacs.svg"> Emacs</button></a>
    <a href="/plugins/sublimetext"><button class="btn btn-primary"><img style="height: 1em;" src="/images/editors/sublime.png"> Sublime</button></a>
    <a href="/plugins/atom"><button class="btn btn-primary"><img style="height: 1em;" style="height: 1em;" src="/images/editors/atom.png"> Atom</button></a>
    <button data-toggle="popover" data-placement="top" data-content="Vim support is not yet implemented" type="button" class="btn btn-default btn-disabled">
      <img class="desaturate" style="height: 1em;" src="/images/editors/vim.svg"> Vim
    </button>

    <p>Interested in building a plugin for an editor srclib doesn't yet
    support? <a target="_blank" href="https://twitter.com/srclib">Let us
    know</a>&mdash;we'd love to help!</p><br>

  </div>
</div>

## Contributing
If you want to start hacking on srclib or write your own srclib toolchain, [join the srclib Slack](http://slackin.srclib.org) and then access it on [srclib.slack.com](https://srclib.slack.com).
<br>
We are more than happy to meet new contributors and to help people to get started on srclib hacking.
