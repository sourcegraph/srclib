MAKEFLAGS+=--no-print-directory

.PHONY: default install src release

default: install

install: src

src: ${GOBIN}/src

${GOBIN}/src: $(shell find -type f -and -name '*.go')
	go install ./cmd/src

release: src
	@mkdir -p releases
	cp ${GOBIN}/src releases/src
	gh release create -p -a releases ${TAG}
