page_title: Overview

# Overview

> The API that `src` will expose has yet to be determined.

The srclib API will be used through the invocation of the `src` executable.
Eventually, we may use a persistent `src` executable that provides a REST-based
web service.

## Quick-start
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

Keep reading to learn more about how srclib models code.
