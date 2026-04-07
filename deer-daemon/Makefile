BINARY_NAME=deer-daemon
BUILD_DIR=bin

# PostHog key (empty by default for dev builds)
POSTHOG_KEY ?=

# Build flags
LDFLAGS=
ifneq ($(POSTHOG_KEY),)
  LDFLAGS += -X github.com/aspectrr/deer.sh/deer-daemon/internal/telemetry.posthogAPIKey=$(POSTHOG_KEY)
endif

.PHONY: all build build-dev run clean fmt vet lint test test-coverage check deps tidy install help redpanda-e2e-lima redpanda-e2e-lima-dry-run

all: fmt vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/deer-daemon

build-dev: POSTHOG_KEY=phc_QR3I1IKrEOqx5jIfJkBMfyznynIxRYd8kzmZM9o9fRZ
build-dev: build ## Build with PostHog key

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run --allow-parallel-runners ./...

test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

check: fmt vet lint test

deps:
	go mod download

tidy:
	go mod tidy

install:
	go install ./cmd/deer-daemon

help:
	@echo "Available targets:"
	@echo "  all            - Run fmt, vet, test, and build (default)"
	@echo "  build          - Build the deer-daemon binary"
	@echo "  run            - Build and run the daemon"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  check          - Run all code quality checks"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy and verify dependencies"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  redpanda-e2e-lima        - Start/create a Lima VM and run the Redpanda live guest E2E test"
	@echo "  redpanda-e2e-lima-dry-run - Print the Lima E2E commands without running them"
	@echo "  help           - Show this help message"

redpanda-e2e-lima:
	bash ../scripts/run-redpanda-e2e-lima-host.sh --repo-root $(abspath ..)

redpanda-e2e-lima-dry-run:
	bash ../scripts/run-redpanda-e2e-lima-host.sh --repo-root $(abspath ..) --dry-run
