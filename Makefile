DOCKER_SOCKET = /var/run/docker.sock
TESTING_DB_NETWORK = test_network5
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"

# testing-with-database-network
# checks

verify: test
checks: errcheck go-mod-check check-imports check-fmt

test:
	go test ./...

test-ci:


linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
	golangci-lint --version