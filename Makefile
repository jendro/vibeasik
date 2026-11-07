.PHONY: hooks build test vet fmt

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

hooks:
	git config core.hooksPath .githooks

build:
	go build -ldflags "-X github.com/vibeasik/vibeasik/pkg/cli.Version=$(VERSION)" -o bin/vibeasik ./cmd/vibeasik

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .
