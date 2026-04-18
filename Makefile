VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY = recap

.PHONY: build install test lint clean fmt

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/recap/

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/recap/

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

fmt:
	go fmt ./...
