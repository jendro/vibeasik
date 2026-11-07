.PHONY: hooks build test vet fmt

hooks:
	git config core.hooksPath .githooks

build:
	go build -o bin/vibeasik ./cmd/vibeasik

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .
