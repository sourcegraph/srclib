# srcgraph testing

This package contains a test for `srcgraph make` output on various repositories.
The test compares the actual and expected output of `srcgraph make` on these
repositories.

The repositories are stored as git submodules in `testdata/repos`. If you
attempt to run tests and have not initialized the submodules, the tests will
automatically run `git submodule init`. If a git submodule's pointer has been
updated but the version in your local checkout hasn't yet been updated, then the
tests will automatically run `git submodule update` on that submodule. (This
will not destroy any local changes you've made.)

The expected output is in `testdata/repos-output/${reponame}/want`. The actual
output is written to the sibling dir `got` (which is not committed to the git
repository).

## Running tests

Run `godep go test`.


## Adding a test case

Add a git submodule to `testdata/repos`. For example:

```
git submodule add git@github.com:sgtest/go-sample-0.git testdata/repos/go-sample-0
```


## Updating expected test output

Run `godep go test -test.mode=gen`.
