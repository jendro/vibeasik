.PHONY: hooks build studio release test vet fmt

VERSION ?= $(shell git describe --tags --match 'v[0-9]*' --always --dirty 2>/dev/null || echo dev)

hooks:
	git config core.hooksPath .githooks

build:
	go build -ldflags "-X github.com/vibeasik/vibeasik/pkg/cli.Version=$(VERSION)" -o bin/vibeasik ./cmd/vibeasik

# Build Studio (Nuxt) lalu salin hasilnya ke pkg/ui/dist agar ikut tersemat di binary.
studio:
	cd studio && bun install --frozen-lockfile && bun run generate
	find pkg/ui/dist -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -r studio/.output/public/. pkg/ui/dist/

release: studio build

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .
