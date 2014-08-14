MAKEFLAGS+=--no-print-directory

.PHONY: default install src release upload-release check-release install-std-toolchains test-std-toolchains

default: install

install: src

src: ${GOBIN}/src

${GOBIN}/src: $(shell find -type f -and -name '*.go')
	go install ./cmd/src

EQUINOX_APP=ap_BQxVz1iWMxmjQnbVGd85V58qz6
release: upload-release check-release

upload-release:
	@bash -c 'if [[ "$(V)" == "" ]]; then echo Must specify version: make release V=x.y.z; exit 1; fi'
	git tag v$(V)
	@equinox release \
	  --platforms 'darwin_amd64 linux_amd64 linux_386' \
	  --private-key ~/.equinox/update.key \
	  --equinox-account $(EQUINOX_ACCOUNT) \
	  --equinox-secret $(EQUINOX_SECRET) \
	  --equinox-app $(EQUINOX_APP) \
	  --version=$(V) \
	  -- \
	  -ldflags "-X sourcegraph.com/sourcegraph/srclib/src.Version $(V)" \
	  cmd/src/src.go
	git push --tags

check-release:
	@bash -c 'if [[ "$(V)" == "" ]]; then echo Must specify version: make release V=x.y.z; exit 1; fi'
	@rm -rf /tmp/src-$(V)
	curl -o /tmp/src-$(V).zip "https://api.equinox.io/1/Applications/$(EQUINOX_APP)/Updates/Asset/src-$(V).zip?os=$(shell go env GOOS)&arch=$(shell go env GOARCH)&channel=stable"
	cd /tmp && unzip src-$(V).zip && chmod +x src-$(V)
	echo; echo
	/tmp/src-$(V) version --no-check-update
	echo; echo
	@echo Released src $(V)

install-std-toolchains:
	src toolchain install-std

test-std-toolchains:
	src toolchain list | grep srclib-go
	src toolchain list | grep srclib-javascript
#	src toolchain list | grep srclib-python
	src toolchain list | grep srclib-ruby

	@echo
	@echo
	@echo Testing installation of standard toolchains in Docker if Docker is running
	(docker info && make -C integration test) || echo Docker is not running...skipping integration tests.
