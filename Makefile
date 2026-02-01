# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/seu-user/header-route-controller:latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=controller

# Build flags
LDFLAGS=-ldflags "-w -s"

.PHONY: all build clean test coverage docker-build docker-push run install uninstall help

all: test build

## Build
build: ## Build the controller binary
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/manager

build-linux: ## Build for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/manager

## Test
test: ## Run tests
	$(GOTEST) -v ./...

coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Clean
clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## Dependencies
deps: ## Download dependencies
	$(GOMOD) download

tidy: ## Tidy go.mod
	$(GOMOD) tidy

## Docker
docker-build: ## Build docker image
	docker build -t $(IMG) .

docker-push: ## Push docker image
	docker push $(IMG)

docker-build-push: docker-build docker-push ## Build and push docker image

## Kubernetes
install: ## Install CRD and RBAC
	kubectl apply -f config/crd/headerroute.yaml
	kubectl apply -f config/rbac/rbac.yaml

uninstall: ## Uninstall CRD and RBAC
	kubectl delete -f config/rbac/rbac.yaml --ignore-not-found
	kubectl delete -f config/crd/headerroute.yaml --ignore-not-found

## Run
run: ## Run controller locally
	$(GOCMD) run ./cmd/manager

## Help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
