# Frequently asked questions

#### Why are there 2 different execution schemes for tools: running directly as
     a program and running inside a Docker container?

Running `src` as a normal program means that it'll pick up context from your
local system, such as interpreter/compiler versions, dependencies, etc. This is
desirable when you're using `src` to analyze code that you're editing locally,
and when you don't want the analysis output to be reproducible by other people.

Running `src` inside Docker means that analysis occurs in a reproducible
environment, with fixed versions of interpreters/compilers and dependencies.
This is desirable when you want to share analysis output with others, such as
when you're uploading it to an external service (like
[Sourcegraph](https://sourcegraph.com)).


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


## Misc.

These are old Q&A that are no longer directly relevant. I've kept them here in
case they might be useful.

TODO(sqs): remove these before release.

#### Why might we want to use Docker's `ADD` in some cases and volumes in another?

There are performance tradeoffs for using volumes vs. `ADD`:

* `ADD` requires sending the whole build context to the server and copying it to
  disk, which is slow and IO-intensive. But it allows us to use `RUN` to perform
  expensive operations that use the `ADD`ed data and modify the container's
  filesystem (such as `npm install` to install dependencies). These `RUN`
  commands are cached, so if multiple handlers need to run the same commands,
  they'll only have to be executed once.
* Volumes don't require sending build context or even copying it at all, so they
  are much faster to build. They also let us simply run the Docker image with
  the files we want instead of having to create another Dockerfile to `ADD`
  those files. However, if multiple handlers run the same expensive commands,
  and if those commands depend on the project's files, then using volumes
  requires us to run them each time (we can't use the Docker build cache).


#### Why can't a tool's handlers use different Docker images from each other?

The Docker image built for a tool should be capable of running all of the tool's
functionality. It would add a lot of complexity to either:

* allow tools to contain multiple Dockerfiles (some of which would probably be
  `FROM` others); or
* allow handlers to generate new Dockerfiles (and then build them with `docker
build - < context.tar`) or run sub-Docker containers.

If a handler truly can't reuse the scanner's Dockerfile, then move it to a
separate tool.

TODO(sqs): Consider adding templating to the root Dockerfile so we can
substitute simple parameters, like `{{.PythonVersion}}`.
