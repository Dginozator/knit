# Makefile for E2EE Messenger

.PHONY: build test clean install run keygen init send receive help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=nit
MAIN_PATH=./cmd/messenger

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux $(MAIN_PATH)

# Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin $(MAIN_PATH)

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME).exe $(MAIN_PATH)

# Build for all platforms
build-all: build-linux build-darwin build-windows

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux
	rm -f $(BINARY_NAME)-darwin
	rm -f $(BINARY_NAME).exe
	rm -f coverage.out coverage.html

# Install dependencies
install:
	$(GOMOD) install

# Run the application
run:
	$(GOCMD) run $(MAIN_PATH)/main.go

# Generate identity
keygen:
	$(GOCMD) run $(MAIN_PATH)/main.go keygen

# Initialize stream
init:
	$(GOCMD) run $(MAIN_PATH)/main.go init

# Send message
send:
	$(GOCMD) run $(MAIN_PATH)/main.go send

# Receive messages
receive:
	$(GOCMD) run $(MAIN_PATH)/main.go receive

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	$(GOCMD) fmt ./...

# Verify module integrity
verify:
	$(GOMOD) verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  build-linux    - Build for Linux"
	@echo "  build-darwin   - Build for macOS"
	@echo "  build-windows  - Build for Windows"
	@echo "  build-all      - Build for all platforms"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  deps           - Download dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install dependencies"
	@echo "  run            - Run the application"
	@echo "  keygen         - Generate identity"
	@echo "  init           - Initialize stream"
	@echo "  send           - Send message"
	@echo "  receive        - Receive messages"
	@echo "  lint           - Lint the code"
	@echo "  fmt            - Format the code"
	@echo "  verify         - Verify module integrity"
