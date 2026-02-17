BINARY_NAME=api
BUILD_DIR=bin

.PHONY: all build run clean fmt vet lint test test-coverage check deps tidy install help

all: fmt vet test build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)

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
	go install ./cmd/server

help:
	@echo "Available targets:"
	@echo "  all            - Run fmt, vet, test, and build (default)"
	@echo "  build          - Build the API binary"
	@echo "  run            - Build and run the API"
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
	@echo "  help           - Show this help message"
