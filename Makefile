
ROOT_PATH = ../../..

NOMS_GO_PACKAGES = github.com/attic-labs/noms/cmd \
	github.com/attic-labs/noms/samples/go \
	github.com/attic-labs/noms/go

NOMS_GIT_REV = $(shell git describe --always)

.PHONY: help clean build install test \
	clean-go build-go install-go test-go \
	build-go-in-dir

help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  test                    to run all the tests"
	@echo "  build                   to build the packages"
	@echo "  clean                   to clean the build and install"
	@echo "  install                 to install the packages"

clean: clean-go

clean-go:
	@go clean -i $(foreach dir,$(NOMS_GO_PACKAGES),$(dir)/...)

build: build-go

build-go:
	@(cd $(ROOT_PATH) && go build -ldflags "-X github.com/attic-labs/noms/go/constants.NomsGitSHA=$(NOMS_GIT_REV)" $(foreach dir,$(NOMS_GO_PACKAGES),$(dir)/...))

build-go-in-dir:
	@echo Building $(NOMS_GIT_REV)
	@for x in $(foreach dir,$(NOMS_GO_PACKAGES),$(wildcard $(ROOT_PATH)/$(dir)/*/)) ; do \
		(cd $$x && echo $$x && go build -ldflags "-X github.com/attic-labs/noms/go/constants.NomsGitSHA=$(NOMS_GIT_REV)") ;\
	done

install: install-go

install-go:
	@(cd $(ROOT_PATH) && go install -ldflags "-X github.com/attic-labs/noms/go/constants.NomsGitSHA=$(NOMS_GIT_REV)" $(foreach dir,$(NOMS_GO_PACKAGES),$(dir)/...))

test: test-go

test-go:
	@go test $(foreach dir,$(NOMS_GO_PACKAGES),$(dir)/...)
