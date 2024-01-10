APP_NAME = kyma-environment-broker
APP_PATH = components/kyma-environment-broker
APP_CLEANUP_NAME = kyma-environments-cleanup-job
APP_SUBACCOUNT_CLEANUP_NAME = kyma-environment-subaccount-cleanup-job
APP_SUBSCRIPTION_CLEANUP_NAME = kyma-environment-subscription-cleanup-job
APP_TRIAL_CLEANUP_NAME = kyma-environment-trial-cleanup-job
DOCKER_SOCKET = /var/run/docker.sock
TESTING_DB_NETWORK = test_network5
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"

# testing-with-database-network
# checks

verify: test
checks: errcheck mod-verify go-mod-check check-imports check-fmt

.PHONY: test
test:
	go test ./...

errcheck:
	#errcheck -blank -asserts -ignorepkg '$$($(DIRS_TO_CHECK) | tr '\n' ',')' -ignoregenerated ./...

check-imports:
	@if [ -n "$$(goimports -l $$($(FILES_TO_CHECK)))" ]; then \
		echo "✗ some files are not properly formatted or contain not formatted imports. To repair run make imports"; \
		goimports -l $$($(FILES_TO_CHECK)); \
		exit 1; \
	fi;

check-fmt:
	@if [ -n "$$(gofmt -l $$($(FILES_TO_CHECK)))" ]; then \
		gofmt -l $$($(FILES_TO_CHECK)); \
		echo "✗ some files are not properly formatted. To repair run make fmt"; \
		exit 1; \
	fi;

mod-verify:
	GO111MODULE=on go mod verify

go-mod-check:
	@echo make go-mod-check
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;