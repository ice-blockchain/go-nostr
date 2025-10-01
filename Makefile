.PHONY: all
all:

.PHONY: test
test:
	go tool gotestsum -f github-actions --packages="./..."

.PHONY: up
up:
	go get -v -u ./...
	go mod tidy
