################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.20
DOCKER_BASE_IMAGE = alpine:3.19

################################################################################

GO ?= $(shell command -v go 2> /dev/null)
ELROND_IMAGE_REPO ?=mattermost/elrond
ELROND_IMAGE ?= mattermost/elrond:test
APP := elrond
MACHINE = $(shell uname -m)
GOFLAGS ?= $(GOFLAGS:)
BUILD_TIME := $(shell date -u +%Y%m%d.%H%M%S)
BUILD_HASH := $(shell git rev-parse HEAD)

################################################################################

LOGRUS_URL := github.com/sirupsen/logrus

LOGRUS_VERSION := $(shell find go.mod -type f -exec cat {} + | grep ${LOGRUS_URL} | awk '{print $$NF}')

LOGRUS_PATH := $(GOPATH)/pkg/mod/${LOGRUS_URL}\@${LOGRUS_VERSION}

export GO111MODULE=on

all: check-style dist

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: govet lint
	@echo Checking for style guide compliance

## Runs lint against all packages.
.PHONY: lint
lint:
	@echo Running lint
	env GO111MODULE=off $(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status ./...
	@echo lint success

## Runs govet against all packages.
.PHONY: vet
govet:
	@echo Running govet
	$(GO) vet ./...
	@echo Govet success

## Builds and thats all :)
.PHONY: dist
dist:	build

.PHONY: binaries
binaries: ## Build binaries of elrond
	@echo Building binaries of elrond
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/elrond-linux-amd64  ./cmd/$(APP)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/elrond-darwin-amd64  ./cmd/$(APP)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/elrond-linux-arm64 ./cmd/$(APP)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/elrond-darwin-arm64  ./cmd/$(APP)


.PHONY: build
build: ## Build the elrond
	@echo Building Elrond for ARCH=$(ARCH)
	@if [ "$(ARCH)" = "amd64" ]; then \
		export GOARCH="amd64"; \
	elif [ "$(ARCH)" = "arm64" ]; then \
		export GOARCH="arm64"; \
	elif [ "$(ARCH)" = "arm" ]; then \
		export GOARCH="arm"; \
	else \
		echo "Unknown architecture $(ARCH)"; \
		exit 1; \
	fi; \
	GOOS=linux CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/elrond  ./cmd/$(APP)

.PHONY: build-image
build-image:  ## Build the docker image for Elrond
	@echo Building Elrond Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(ELROND_IMAGE) \
	--no-cache \
	--push

.PHONY: build-image-with-tag
build-image-with-tag:  ## Build the docker image for elrond
	@echo Building Elrond Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	: $${TAG:?}
	docker buildx build \
	--platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(ELROND_IMAGE) -t $(ELROND_IMAGE_REPO):${TAG} \
	--push

.PHONY: push-image-pr
push-image-pr:
	@echo Push Image PR
	./scripts/push-image-pr.sh

.PHONY: push-image
push-image:
	@echo Push Image
	./scripts/push-image.sh

.PHONY: install
install: build
	go install ./...

.PHONY: release
release:
	@echo Cut a release
	./scripts/release.sh

.PHONY: deps
deps:
	apt-get update && apt-get install -y hub git
	go install k8s.io/release/cmd/release-notes@latest

.PHONY: unittest
unittest:
	$(GO) test ./... -v -covermode=count -coverprofile=coverage.out
