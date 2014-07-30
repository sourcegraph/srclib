FROM ubuntu:14.04

RUN apt-get update -qq
RUN apt-get install -qq golang build-essential git mercurial

ENV GOPATH /srclib
ENV PATH /srclib/bin:$PATH

ADD . /srclib/src/github.com/sourcegraph/srclib/
RUN go get -v github.com/sourcegraph/srclib/...
RUN go install github.com/sourcegraph/srclib/cmd/src
RUN go get -v github.com/sourcegraph/srclib-go
RUN cd /srclib/src/github.com/sourcegraph/srclib && git clone https://github.com/sourcegraph/srclib-javascript

RUN mkdir -p /root/.srclib/github.com/sourcegraph
RUN ln -rs /srclib/src/github.com/sourcegraph/srclib-go /root/.srclib/github.com/sourcegraph/srclib-go
RUN ln -rs /srclib/src/github.com/sourcegraph/srclib-javascript /root/.srclib/github.com/sourcegraph/srclib-javascript

ENTRYPOINT /srclib/bin/srclib
