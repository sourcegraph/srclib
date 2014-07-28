# Frequently asked questions

#### Why are there 2 different execution schemes for tools: running directly as a program and running inside a Docker container?

Running toolchains as normal programs means that it'll pick up context from your
local system, such as interpreter/compiler versions, dependencies, etc. This is
desirable when you're using `src` to analyze code that you're editing locally,
and when you don't want the analysis output to be reproducible by other people.

Running toolchains inside Docker means that analysis occurs in a reproducible
environment, with fixed versions of interpreters/compilers and dependencies.
This is desirable when you want to share analysis output with others, such as
when you're uploading it to an external service (like
[Sourcegraph](https://sourcegraph.com)).



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
  commands are cached, so if multiple tools need to run the same commands,
  they'll only have to be executed once.
* Volumes don't require sending build context or even copying it at all, so they
  are much faster to build. They also let us simply run the Docker image with
  the files we want instead of having to create another Dockerfile to `ADD`
  those files. However, if multiple tools run the same expensive commands, and
  if those commands depend on the project's files, then using volumes requires
  us to run them each time (we can't use the Docker build cache).


#### Why can't a toolchain's tools use different Docker images from each other?

The Docker image built for a toolchain should be capable of running all of the
toolchain's functionality. It would add a lot of complexity to either:

* allow toolchains to contain multiple Dockerfiles (some of which would probably
  be `FROM` others); or
* allow tools to generate new Dockerfiles (and then build them with `docker
  build - < context.tar`) or run sub-Docker containers.

If a tool truly can't reuse the scanner's Dockerfile, then move it to a separate
toolchain.

TODO(sqs): Consider adding templating to the root Dockerfile so we can
substitute simple parameters, like `{{.PythonVersion}}`.
