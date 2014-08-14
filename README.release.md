# srclib release process

We use [Equinox](https://equinox.io) to release cross-compiled Go binaries for a
variety of platforms.

Releases are signed using a private key. Committers with release privileges
should already have this private key.

To issue a new release:

1. Update `const Version` in `src/cli.go` to the new version number.
1. Commit the change.
1. Tag the commit as `vN.N.N`, where `N.N.N` is the value of `const Version`.
1. Push the tag with `git push --tags`.
1. Build and upload new binaries:
   ```
   equinox release --platforms 'darwin_amd64 linux_amd64 linux_386' --private-key PATH_TO_UPDATE_KEY --equinox-account id_ACCOUNT --equinox-secret key_SECRET --equinox-app ap_BQxVz1iWMxmjQnbVGd85V58qz6 --version=N.N.N cmd/src/src.go
   ```

Note: to cross-compile Go binaries, you'll have to perform a one-time setup step
in your GOROOT:

```
cd $GOROOT/src
GOOS=darwin GOARCH=amd64 ./make.bash --no-clean
GOOS=linux GOARCH=386 ./make.bash --no-clean
GOOS=linux GOARCH=amd64 ./make.bash --no-clean
```

You can omit the command that contains your current `GOOS` and `GOARCH`, as
you've already bootstrapped that combo.

Users of `src` can check for updates with `src version` and update the program
with `src selfupdate`.
