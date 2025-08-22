# Topi Multi-Module Project Root Makefile
# This is the orchestrator Makefile that delegates to individual module Makefiles

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# CONTAINER_TOOL defines the container tool to be used for building images.
CONTAINER_TOOL ?= docker

# Define modules
MODULES := shared engine scheduler builder

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt-all
fmt-all: ## Run go fmt against all modules.
	@for module in $(MODULES); do \
		echo "Formatting $$module..."; \
		$(MAKE) -C $$module fmt; \
	done

.PHONY: vet-all
vet-all: ## Run go vet against all modules.
	@for module in $(MODULES); do \
		echo "Vetting $$module..."; \
		$(MAKE) -C $$module vet; \
	done

.PHONY: test-all
test-all: ## Run tests for all modules.
	@for module in $(MODULES); do \
		echo "Testing $$module..."; \
		$(MAKE) -C $$module test; \
	done

.PHONY: lint-all
lint-all: ## Run golangci-lint for all modules.
	@for module in $(MODULES); do \
		if [ "$$module" != "shared" ]; then \
			echo "Linting $$module..."; \
			$(MAKE) -C $$module lint; \
		fi; \
	done

.PHONY: mod-download-all
mod-download-all: ## Download go modules for all modules.
	@for module in $(MODULES); do \
		echo "Downloading modules for $$module..."; \
		$(MAKE) -C $$module mod-download; \
	done

.PHONY: mod-tidy-all
mod-tidy-all: ## Tidy go modules for all modules.
	@for module in $(MODULES); do \
		echo "Tidying modules for $$module..."; \
		$(MAKE) -C $$module mod-tidy; \
	done

##@ Build

.PHONY: build-all
build-all: ## Build all module binaries.
	@echo "Building engine..."
	$(MAKE) -C engine build
	@echo "Building scheduler..."
	$(MAKE) -C scheduler build
	@echo "Building builder..."
	$(MAKE) -C builder build

.PHONY: docker-build-all
docker-build-all: ## Build Docker images for all modules.
	@echo "Building engine Docker image..."
	$(MAKE) -C engine docker-build
	@echo "Building scheduler Docker image..."
	$(MAKE) -C scheduler docker-build
	@echo "Building builder Docker image..."
	$(MAKE) -C builder docker-build

.PHONY: docker-push-all
docker-push-all: ## Push Docker images for all modules.
	@echo "Pushing engine Docker image..."
	$(MAKE) -C engine docker-push
	@echo "Pushing scheduler Docker image..."
	$(MAKE) -C scheduler docker-push
	@echo "Pushing builder Docker image..."
	$(MAKE) -C builder docker-push

.PHONY: kind-push-all
kind-push-all: ## Build and push all Docker images to local kind registry.
	@echo "Building and pushing engine to kind registry..."
	$(MAKE) -C engine kind-push
	@echo "Building and pushing scheduler to kind registry..."
	$(MAKE) -C scheduler kind-push
	@echo "Building and pushing builder to kind registry..."
	$(MAKE) -C builder kind-push

.PHONY: kind-deploy-scheduler
kind-deploy-scheduler: ## Deploy scheduler to kind cluster using local registry.
	$(MAKE) -C scheduler kind-deploy

##@ Run

.PHONY: run-engine
run-engine: ## Run engine from your host.
	$(MAKE) -C engine run

.PHONY: run-scheduler
run-scheduler: ## Run scheduler from your host.
	$(MAKE) -C scheduler run

.PHONY: run-builder
run-builder: ## Run builder from your host.
	$(MAKE) -C builder run

##@ Environment

.PHONY: dev-env-up
dev-env-up: ## Start development environment (PostgreSQL, RabbitMQ, Gitea).
	docker-compose up -d

.PHONY: dev-env-down
dev-env-down: ## Stop development environment.
	docker-compose down

.PHONY: dev-env-logs
dev-env-logs: ## Show logs from development environment.
	docker-compose logs -f

##@ Kubernetes

.PHONY: install-crds
install-crds: ## Install CRDs into the K8s cluster.
	$(MAKE) -C scheduler install

.PHONY: uninstall-crds
uninstall-crds: ## Uninstall CRDs from the K8s cluster.
	$(MAKE) -C scheduler uninstall

.PHONY: deploy-scheduler
deploy-scheduler: ## Deploy scheduler to the K8s cluster.
	$(MAKE) -C scheduler deploy

.PHONY: undeploy-scheduler
undeploy-scheduler: ## Undeploy scheduler from the K8s cluster.
	$(MAKE) -C scheduler undeploy

##@ Workspace

.PHONY: workspace-sync
workspace-sync: ## Sync go workspace.
	go work sync

.PHONY: clean
clean: ## Clean build artifacts from all modules.
	@for module in $(MODULES); do \
		if [ -d "$$module/bin" ]; then \
			echo "Cleaning $$module/bin..."; \
			rm -rf $$module/bin; \
		fi; \
	done

##@ Testing

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests.
	$(MAKE) -C scheduler test-e2e
