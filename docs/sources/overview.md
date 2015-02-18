no_toc: true

# ![srclib symbol](/images/srclib_symbol.svg) **srclib** is a hackable, multi-language code analysis library for building better code tools.

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
<a href="api/overview.md"><strong>common output format</strong></a>, and tools (such as <a href="plugins/TODO"><strong>editor plugins</strong></a>) that
consume this format.
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
          <li><a href="https://sourcegraph.com/sourcegraph/emacs-sourcegraph-mode">Emacs</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/sourcegraph-sublime">Sublime</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/sourcegraph-atom">Atom</a></li>
          <li><a href="https://github.com/lazywei/vim-sourcegraph">Vim (WIP)</a></li>
          <li><a href="#TODO" class="contribute">Contribute a plugin</a></li>
          <li>&nbsp;</li>
        </ul>
      </div><!--
      --><div>
        <label>Languages:</label>
        <ul class="list-unstyled">
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-go">Go</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-java">Java</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-python">Python</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-haskell">JavaScript</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-haskell">Haskell</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-haskell">Ruby (WIP)</a></li>
          <li><a href="https://sourcegraph.com/sourcegraph/srclib-haskell">PHP (WIP)</a></li>
          <li><a href="#TODO" class="contribute">Contribute a new language</a></li>
        </ul>
      </div>
    </div>
  </li>
  <li>
    <label>View code on:</label>
    <a class="btn btn-sm btn-default" target="_blank" href="https://sourcegraph.com/sourcegraph/srclib">&#x2731; Sourcegraph</button></a><!--
    --><a class="btn btn-sm btn-default" target="_blank" href="https://github.com/sourcegraph/srclib"><i class="fa fa-github"></i> GitHub</a>
  </li>
</ul>

</div>
</div>
