<<<<<<< HEAD
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
=======
APP_NAME = kyma-environment-broker
APP_PATH = components/kyma-environment-broker
APP_CLEANUP_NAME = kyma-environments-cleanup-job
APP_SUBACCOUNT_CLEANUP_NAME = kyma-environment-subaccount-cleanup-job
APP_TRIAL_CLEANUP_NAME = kyma-environment-trial-cleanup-job
>>>>>>> main

GOLINT_VER = "v1.55.2"
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"
VERIFY_IGNORE := /vendor\|/automock

 ## The headers are represented by '##@' like 'General' and the descriptions of given command is text after '##''.
.PHONY: help
help: 
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ General

.PHONY: verify
verify: test checks go-lint ## verify simulates same behaviour as 'verify' GitHub Action which run on PR

.PHONY: checks
checks: check-go-mod-tidy check-go-fmt check-gqlgen check-go-imports ## run different go checks

.PHONY: test 
test: ## run Go tests
	go test ./...

.PHONY: go-lint
go-lint: ## linter config in file at root of project -> '.golangci.yaml'
	@if ! [ "$(command -v golangci-lint version --format short)" == $GOLINT_VER ]; then \
  		echo golangci in version $(GOLINT_VER) not found. will be downloaded; \
		GOBIN= go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLINT_VER); \
		echo golangci installed in $(GOBIN) with version: $(shell golangci-lint version --format short); \
	fi;
	golangci-lint run ./...

.PHONY: check-go-mod-tidy
check-go-mod-tidy: ## check if go mod tidy needed
	@echo check-go-mod-tidy
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;

## TODO: Replace by using golangci-lint configuration
.PHONY: check-go-fmt ## run Go fmt against changes
check-go-fmt:
	@echo check-go-fmt
	@if [ -n "$$(gofmt -l $$($(FILES_TO_CHECK)))" ]; then \
		gofmt -l $$($(FILES_TO_CHECK)); \
		echo "✗ some files are not properly formatted. To repair run make fmt"; \
		exit 1; \
	fi;

.PHONY: check-go-imports ## run Go imports against changes
check-go-imports:
	@echo check-go-imports
	@if [ -n "$$(goimports -l $$($(FILES_TO_CHECK)))" ]; then \
		echo "✗ some files are not properly formatted or contain not formatted imports. To repair run make imports"; \
		goimports -l $$($(FILES_TO_CHECK)); \
		exit 1; \
	fi;

.PHONY: check-gqlgen
check-gqlgen: ## run GraphQL changes
	@echo check-gqlgen
	@if [ -n "$$(git status -s pkg/graphql)" ]; then \
		echo -e "${RED}✗ gqlgen.sh modified some files, schema and code are out-of-sync${NC}"; \
		git status -s pkg/graphql; \
		exit 1; \
	fi;