.PHONY: ⚙️  # make all non-file targets phony

.DEFAULT_GOAL := all

BINARY  := paperclip-go
CMD     := ./cmd/paperclip-go
OUT     := ./bin/$(BINARY)

include scripts/help.mk

all: ⚙️ build test lint  ## Build, test, and lint the project

build: ⚙️  ## Build the paperclip-go binary
	go build -ldflags "-X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o "$(OUT)" "$(CMD)"

run: ⚙️  ## Run the paperclip-go server
	go run "$(CMD)" serve

test: ⚙️  ## Run tests
	go test ./...

lint: ⚙️  ## Run linters
	go vet ./...

clean: ⚙️  ## Clean up build artifacts
	rm -rf bin/
