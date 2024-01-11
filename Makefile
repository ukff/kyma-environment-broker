DOCKER_SOCKET = /var/run/docker.sock
TESTING_DB_NETWORK = test_network5
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"

verify: test checks
checks: go-mod-tidy-check

test:
	go test ./...

go-mod-tidy-check:
	@echo make go-mod-check
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;