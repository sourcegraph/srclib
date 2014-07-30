# Overview
## Quick-start
Start by installing the `src` executable, with the following command:

```bash
go get github.com/sourcegraph/srclib/cmd/src
```

Clone a repository in a supported language, for example:
```bash
git clone https://github.com/mitsuhiko/flask
```

Navigate into the repository and invoke `src make`.
```bash
cd flask
src make
```

This should build the repository graph - once complete, you can `cd` into a directory
called `.sourcegraph-data`. In that folder, you should find several files.

Keep reading to learn more about how Srclib models code.
