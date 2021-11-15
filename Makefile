.PHONY: help all format test fmt vet lint golint errcheck vendor-update build clean install-errcheck install-golint

NAME := sisyphus
GO_VER := 1.17.3-buster
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

setup:
	docker pull golang:$(GOVER)

test: fmt lint ## run tests
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go test -race -mod=vendor ./...

fmt: ## only run gofmt
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) gofmt -l -w -s *.go

lint: golint errcheck ## run all linting steps

golint: install-golint ## run golint against code
	$(CURDIR)/go/bin/golint .

errcheck: install-errcheck ## run errcheck against code
	$(CURDIR)/go/bin/errcheck -verbose -exclude errcheck_excludes.txt -asserts -blank -tags static,netgo -mod=readonly ./...

vendor-update: ## update vendor dependencies
	rm -rf go.mod go.sum vendor/
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod tidy -compat=1.17
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod tidy
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go mod vendor

build: ## actually build package
	docker run --rm -v $(CURDIR):/app:z -w /app golang:$(GO_VER) go build -tags static,netgo -mod=vendor -ldflags="-X 'main.Version=$(PKG_TAG)' -X 'main.BuildUser=$(BUILDUSER)' -X 'main.BuildTime=$(BUILDTIME)'" -o $(NAME) .

clean: ## remove build artifacts
	rm -f $(NAME)

install-golint: ## install golint binary
	test -f $(CURDIR)/go/bin/golint || GOPATH=$(CURDIR)/go/ GO111MODULE=off go get -u golang.org/x/lint/golint

install-errcheck: ## install errcheck binary
	test -f $(CURDIR)/go/bin/errcheck || GOPATH=$(CURDIR)/go/ GO111MODULE=off go get -u github.com/kisielk/errcheck
