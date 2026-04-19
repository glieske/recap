VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY = recap

.PHONY: build install test lint clean fmt release

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

release:
	@read -p "Version to release (e.g. v1.2.3): " VERSION_INPUT; \
	if ! echo "$$VERSION_INPUT" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: version must match vX.Y.Z (e.g. v1.2.3)"; \
		exit 1; \
	fi; \
	read -p "Create tag $$VERSION_INPUT? [y/N] " CONFIRM_TAG; \
	if [ "$$CONFIRM_TAG" != "y" ] && [ "$$CONFIRM_TAG" != "Y" ]; then \
		echo "Aborted."; \
		exit 0; \
	fi; \
	echo "Running lint..."; \
	go vet ./... || exit 1; \
	echo "Running tests..."; \
	go test ./... || exit 1; \
	git tag "$$VERSION_INPUT"; \
	echo "Tag $$VERSION_INPUT created."; \
	read -p "Push tag $$VERSION_INPUT to origin? [y/N] " CONFIRM_PUSH; \
	if [ "$$CONFIRM_PUSH" != "y" ] && [ "$$CONFIRM_PUSH" != "Y" ]; then \
		echo "Tag created locally but not pushed."; \
		exit 0; \
	fi; \
	git push origin "$$VERSION_INPUT"; \
	echo "Released $$VERSION_INPUT."
