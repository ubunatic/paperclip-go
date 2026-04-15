# Makefile for paperclip-go
BINARY  := paperclip-go
CMD     := ./cmd/paperclip-go
OUT     := ./bin/$(BINARY)

.PHONY: build test lint clean

build:
	go build -o $(OUT) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
