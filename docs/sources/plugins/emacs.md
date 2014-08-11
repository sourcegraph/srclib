# Emacs Plugin

<div class="embed-responsive embed-responsive-16by9">
<iframe class="embed-responsive-item" src="//www.youtube.com/embed/cm59qQD6khs" frameborder="0" allowfullscreen></iframe>
</div>

## Features
- Documentation lookups
- Type information
- Find usages (across all open-source projects globally)

## Installation Instructions
First, make sure you've installed srclib, following the [instructions here](../gettingstarted.md#install-srclib).

Once srclib is installed, you can install the emacs plugin by navigating to your `.emacs.d` directory and cloning the repository.
```bash
cd ~/.emacs.d
git clone https://github.com/sourcegraph/emacs-sourcegraph-mode.git
```

Then you can add a hook, or run add a hook or just run `sourcegraph-mode` manually to enable it in a buffer.

In any file (with sourcegraph-mode enabled), run sourcegraph-describe (or C-M-.) to see docs, type info, and examples.

## Contribute on GitHub
<iframe src="http://ghbtns.com/github-btn.html?user=sourcegraph&repo=emacs-sourcegraph-mode&type=watch&count=true&size=large"
  allowtransparency="true" frameborder="0" scrolling="0" width="170" height="30"></iframe>
