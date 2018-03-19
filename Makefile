SHELL := /bin/bash

all: build

GO := go
TAR ?= tar

ARCH ?= $(shell go env GOARCH)
OS ?= $(shell go env GOOS)
CGO ?= $(shell go env CGO_ENABLED)

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_RECENT_TAG = $(shell git describe --tags --abbrev=0 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_VERSION = $(shell ./hack/version.sh | awk -F': ' '/^VERSION:/ {print $$2}')

OUT_DIR := bin
SRC_PREFIX := qiniu.com/account/app
PKGS = $(shell $(GO) list $(SRC_PREFIX)/...)

# Only set Version if building a tag or VERSION is set
ifneq ($(VERSION),)
    LDFLAGS += -X $(SRC_PREFIX)/version.gitVersion=${VERSION}
else
    LDFLAGS += -X $(SRC_PREFIX)/version.gitVersion=${GIT_VERSION}
endif

# Clear the "unreleased" string in BuildMetadata
ifneq ($(GIT_TAG),)
    LDFLAGS += -X $(SRC_PREFIX)/version.BuildMetadata=
endif
LDFLAGS += -X $(SRC_PREFIX)/version.gitCommit=${GIT_COMMIT}
LDFLAGS += -X $(SRC_PREFIX)/version.gitTreeState=${GIT_DIRTY}
LDFLAGS += -X $(SRC_PREFIX)/version.buildDate=${BUILD_DATE}

TARGET := zk-controller

build:
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=$(CGO) $(GO) build -ldflags "${LDFLAGS}" -o $(OUT_DIR)/$(OS)/$(TARGET) $(SRC_PREFIX)
.PHONY: build

crossbuild: OS = linux
crossbuild: ARCH = amd64
crossbuild: CGO = 0
crossbuild: build
.PHONY: crossbuild

vendor:
	./hack/ensure-deps.sh
.PHONY: vendor

fmt:
	@echo ">>> formatting code"
	@$(GO) fmt $(PKGS)
.PHONY: fmt

vet:
	@$(GO) vet $(PKGS)
.PHONY: vet

test:
	@$(GO) test $(PKGS)
.PHONY: test

testaone:
	curl https://aone.qiniu.io/api/coverage/collect?token=A59B8029-E9A1-4FF3-B72B-3657CEAC64D4 | bash

lint:
	@if ! which golint >/dev/null; then \
		go get -u github.com/golang/lint/golint; \
	fi
	@echo $(PKGS) | xargs -n 1 golint
.PHONY: lint

clean:
	@rm -rf $(OUT_DIR)/*
.PHONY: clean

info:
	@echo "Version:           ${VERSION}"
	@echo "Git Version:       ${GIT_VERSION}"
	@echo "Git Recent Tag:    ${GIT_RECENT_TAG}"
	@echo "Git Commit:        ${GIT_COMMIT}"
	@echo "Git Tree State:    ${GIT_DIRTY}"
	@echo "Build Date:        ${BUILD_DATE}"
.PHONY: info
