export PATH := $(CURDIR)/bin:$(PATH)

.PHONY: build lint test fix generate install-deps migrate-up migrate-down migrate-status migrate-new
.DEFAULT_GOAL := build

build: lint test
	go build -o bin/ .

fix:
	golangci-lint fmt ./...
	golangci-lint run --fix ./...

lint:
	sqlc diff
	golangci-lint run ./...
	go vet ./...
	govulncheck ./...

generate:
	sqlc generate

test:
	go test -race ./...

migrate-up:
	goose up

migrate-down:
	goose down

migrate-status:
	goose status

migrate-new:
	@test -n "$(name)" || { echo "usage: make migrate-new name=<snake_case>"; exit 1; }
	goose create $(name) sql

install-deps:
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	GOBIN=$(CURDIR)/bin go install golang.org/x/vuln/cmd/govulncheck@latest
	GOBIN=$(CURDIR)/bin go install github.com/pressly/goose/v3/cmd/goose@latest
	GOBIN=$(CURDIR)/bin go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
