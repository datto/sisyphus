.PHONY: help all test fmt lint golint errcheck vendor-update build clean install-errcheck install-golint

NAME := sisyphus
GO_VER := 1.20.7
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)
BUILDTIME ?= $(shell date)
BUILDUSER ?= $(shell id -u -n)
PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(shell echo $$(git describe --long --all | tr '/' '-')$$(git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -c 10-17)))
endif

all: test build ## format, lint, and build the package

help:
	@echo "Makefile targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' Makefile \
	| sed -n 's/^\(.*\): \(.*\)##\(.*\)/    \1 :: \3/p' \
	| column -t -c 1  -s '::'

setup: envsetup
	docker pull golang:$(GO_VER)
	docker pull golangci/golangci-lint:latest

envsetup:
	mkdir -p $(CURDIR)/.cache/
	chmod -R 777 $(CURDIR)/.cache/ ## fix for rootless container environments (podman, etc.)

test: envsetup fmt lint ## run tests
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go test -race -mod=vendor ./...

fmt: envsetup ## only run gofmt
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) gofmt -l -w -s *.go

lint: envsetup ## run all linting steps
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golangci/golangci-lint:latest golangci-lint run --sort-results

vendor-update: envsetup ## update vendor dependencies
	rm -rf go.mod go.sum vendor/
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod init $(NAME)
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod tidy
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod vendor

build: envsetup ## actually build package
	docker run --rm --user $(CURRENT_UID):$(CURRENT_GID) -v $(CURDIR)/.cache/:/.cache/ -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go build -tags static,netgo -mod=vendor -ldflags="-X 'main.Version=$(PKG_TAG)' -X 'main.BuildUser=$(BUILDUSER)' -X 'main.BuildTime=$(BUILDTIME)'" -o $(NAME)-bullseye .

clean: ## remove build artifacts
	rm -f $(NAME)
