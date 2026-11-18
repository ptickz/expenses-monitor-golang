export PATH := $(CURDIR)/bin:$(PATH)

.PHONY: build lint test fix install-deps
.DEFAULT_GOAL := build

build: lint test
	go build -o bin/ .

fix:
	golangci-lint fmt ./...
	golangci-lint run --fix ./...

lint:
	golangci-lint run ./...
	go vet ./...
	govulncheck ./...

test:
	go test -race ./...

install-deps:
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	GOBIN=$(CURDIR)/bin go install golang.org/x/vuln/cmd/govulncheck@latest
