GOLINT_VER = v1.55.2

 ## The headers are represented by '##@' like 'General' and the descriptions of given command is text after '##''.
.PHONY: help
help: 
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ General

.PHONY: verify
verify: test checks go-lint ## verify simulates same behaviour as 'verify' GitHub Action which run on every PR

.PHONY: checks
checks: check-go-mod-tidy ## run different Go related checks

.PHONY: go-lint
go-lint: go-lint-install ## linter config in file at root of project -> '.golangci.yaml'
	golangci-lint run --new

go-lint-install: ## linter config in file at root of project -> '.golangci.yaml'
	@if [ "$(shell command golangci-lint version --format short)" != "$(GOLINT_VER)" ]; then \
  		echo golangci in version $(GOLINT_VER) not found. will be downloaded; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLINT_VER); \
		echo golangci installed with version: $(shell command golangci-lint version --format short); \
	fi;
	
##@ Tests

.PHONY: test 
test: ## run Go tests
	go test ./...

##@ Go checks 

.PHONY: check-go-mod-tidy
check-go-mod-tidy: ## check if go mod tidy needed
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;

##@ Development support commands

.PHONY: fix
fix: go-lint-install ## try to fix automatically issues
	go mod tidy
	golangci-lint run --fix