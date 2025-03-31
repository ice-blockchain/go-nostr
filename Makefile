.PHONY: all
all:

.PHONY: test
test:
	go test -v ./...

.PHONY: up
up:
	go get -v -u ./...
	go mod tidy
