# Frequently asked questions

#### Why are there 2 different execution schemes for tools: running directly as
     a program and running inside a Docker container?

Sometimes you want repeatable builds that aren't dependent on what's on your
local system, and sometimes you do.


#### Why can tools exist in 2 places, the PATH and SRCLIBPATH?

As explained above, we want to be able to run tools directly *or* inside Docker
containers. There's a standard way to run any tool in Docker, given a Dockerfile:
just build the Dockerfile and run the image. But there's no standard way for
running other programs.

For example, let's say we have 2 tools in our SRCLIBPATH. One of them is a Go scanner:

```
SRCLIBPATH/github.com/alice/srclib-go/Dockerfile
                                      Srclibtool
                                      golang/scan.go
                                      cmd/src-tool-go/main.go
```

And the other is a Python scanner:


```
SRCLIBPATH/github.com/bob/srclib-python/Dockerfile
                                        Srclibtool
                                        setup.py
                                        requirements.txt
                                        srclib_python/__init__.py
                                        srclib_python/scan.py
                                        bin/src-tool-python.py
```

To run either of these tools in Docker, we just go to its directory and run:

```
docker build -t $IMAGE .
docker run $IMAGE
```

The Dockerfile specifies all of the steps necessary to set up the tool in the
container, including downloading dependencies and installing
interpreters/compilers.

But how do we run them on our local machine? There is no general way.

* For the Go scanner, we need to ensure the Go compiler is installed, set the
  `GOPATH` and `GOBIN` environemnt variables, and then run `go get` and `go
  install`.
* For the Python scanner, we need to install the right Python version, set up
  pip to install binaries in our PATH, and then run `pip install -r
  requirements.txt` and `python setup.py install`.

The Dockerfile typically contains similar steps for building the tool inside the
container, but we can't just run those commands locally because our machine
probably isn't identical to the container's Linux distro, and those commands
might clobber local files/installations.

One possible solution is to require that every tool includes a wrapper script
that transparently builds the tool and execs it, but that is non-portable and
requires a lot of additional work.

So, we just rely on the standard installation scheme for each tool's language or
build system. The tool should include a README that describes how to install it
locally.

Since we rely on existing build systems to install tools, we can't easily
dictate that they be installed to a specific path, such as
`SRCLIBPATH/github.com/bob/srclib-python/bin/src-tool-python.` Every tool could
include a script that copies the installed program to a location like that, but
that's non-standard.

So, we just say that programs installed in your PATH whose names match
`src-tool-*` are srclib tools. It's very easy to configure every build system to
install programs in this way, and it's easy to use the tools as standalone
programs.
