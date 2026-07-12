GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || command -v golangci-lint-v2 2>/dev/null)

.PHONY: all build test vet lint tidy

all: vet lint test

build:
	go build ./...

# Run all tests.
test:
	go test ./...

vet:
	go vet ./...

# Lint with golangci-lint (v2). Also lints the native-Wayland build-tagged files.
lint:
	@if [ -z "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found; install from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run ./...
	$(GOLANGCI_LINT) run --build-tags wayland ./...

tidy:
	go mod tidy
