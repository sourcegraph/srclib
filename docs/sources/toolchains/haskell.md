# Haskell Toolchain
## Installation

This toolchain is not a standalone program; it provides additional functionality
to editor plugins and other applications that use [srclib](https://srclib.org).

Using the Haskell toolchain currently requires [Docker](https://docs.docker.com/installation/#installation).

Make sure [Docker](https://docs.docker.com/installation/#installation) and [srclib](../install.md) are installed, and then run these commands to build the docker image.

```bash
# Fetch the latest code with git.
git clone sourcegraph.com/sourcegraph/srclib-haskell
cd srclib-haskell

# link this toolchain in your SRCLIBPATH (default ~/.srclib) to enable it
srclib toolchain add sourcegraph.com/sourcegraph/srclib-haskell

# Build a docker image that will run the toolchain.
srclib toolchain build sourcegraph.com/sourcegraph/srclib-haskell
```

To verify that the installation succeeded, run:

```
srclib toolchain list
```

You should see this srclib-haskell toolchain in the list.

Now that this toolchain is installed, any program that relies on srclib (such as
editor plugins) will support Haskell.
