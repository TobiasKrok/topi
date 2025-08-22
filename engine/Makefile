# Image URL to use for building/pushing image
IMG ?= topi-engine:latest
# Local kind registry settings
KIND_REGISTRY ?= localhost:5001
KIND_IMG ?= $(KIND_REGISTRY)/topi-engine:latest

# CONTAINER_TOOL defines the container tool to be used for building images.
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -v

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: mod-download
mod-download: ## Download go modules.
	go mod download

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules.
	go mod tidy

##@ Build

.PHONY: build
build: fmt vet ## Build engine binary.
	go build -o bin/engine cmd/engine/main.go

.PHONY: run
run: fmt vet ## Run engine from your host.
	go run ./cmd/engine/main.go

.PHONY: docker-build
docker-build: ## Build docker image.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image.
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: kind-push
kind-push: docker-build ## Build and push docker image to local kind registry.
	$(CONTAINER_TOOL) tag ${IMG} ${KIND_IMG}
	$(CONTAINER_TOOL) push ${KIND_IMG}