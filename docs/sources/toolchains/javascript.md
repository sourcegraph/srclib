# Javascript Toolchain
## Installation

This toolchain is not a standalone program; it provides additional functionality
to editor plugins and other applications that use [srclib](https://srclib.org).

First,
[install the `src` program (see srclib installation instructions)](../gettingstarted.md#install-srclib).

Then run:

```bash
git clone https://github.com/sourcegraph/srclib-javascript.git
cd srclib-javascript
src toolchain add sourcegraph.com/sourcegraph/srclib-javascript
```

To verify that installation succeeded, run:

```
src toolchain list
```

You should see this srclib-javascript toolchain in the list.

Now that this toolchain is installed, any program that relies on srclib (such as
editor plugins) will support JavaScript.
