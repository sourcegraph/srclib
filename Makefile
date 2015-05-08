MAKEFLAGS+=--no-print-directory

.PHONY: default install src release upload-release check-release install-std-toolchains test-std-toolchains

default: install

install: src

src: ${GOBIN}/src

${GOBIN}/src: $(shell find . -type f -and -name '*.go')
	go install ./cmd/src

release: upload-release check-release

SELFUPDATE_TMPDIR=.tmp-selfupdate
upload-release:
	@bash -c 'if [[ "$(V)" == "" ]]; then echo Must specify version: make release V=x.y.z; exit 1; fi'
	go get github.com/laher/goxc github.com/sqs/go-selfupdate
	goxc -q -pv="$(V)"
	go-selfupdate -o="$(SELFUPDATE_TMPDIR)" -cmd=src "release/$(V)" "$(V)"
	aws s3 sync --acl public-read "$(SELFUPDATE_TMPDIR)" s3://srclib-release/src
	git tag v$(V)
	git push --tags

check-release:
	@bash -c 'if [[ "$(V)" == "" ]]; then echo Must specify version: make release V=x.y.z; exit 1; fi'
	@rm -rf /tmp/src-$(V).gz
	curl -Lo /tmp/src-$(V).gz "https://srclib-release.s3.amazonaws.com/src/$(V)/$(shell go env GOOS)-$(shell go env GOARCH)/src.gz"
	cd /tmp && gunzip -f src-$(V).gz && chmod +x src-$(V)
	echo; echo
	/tmp/src-$(V) version
	echo; echo
	@echo Released src $(V)

install-std-toolchains:
	src toolchain install-std

toolchains ?= go javascript python ruby

test-std-toolchains:
	@echo Checking that all standard toolchains are installed
	for lang in $(toolchains); do echo $$lang; src toolchain list | grep srclib-$$lang; done

	@echo
	@echo
	@echo Testing installation of standard toolchains in Docker if Docker is running
	(docker info && make -C integration test) || echo Docker is not running...skipping integration tests.

regen-std-toolchain-tests:
	for lang in $(toolchains); do echo $$lang; cd ~/.srclib/sourcegraph.com/sourcegraph/srclib-$$lang; src test --gen; done
