.PHONY: build test install clean lint release

# Build variables
BINARY_NAME=spectrehub
BUILD_DIR=bin
MAIN_PATH=./cmd/spectrehub

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms (for releases)
release:
	@echo "Building $(BINARY_NAME) for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "Building for macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@echo "Building for macOS arm64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Creating checksums..."
	@cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "Release builds complete:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-*

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -cover ./...

# Run contract tests
test-contract:
	@echo "Running contract tests..."
	$(GOTEST) -v -run TestContract ./...

# Install binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOINSTALL) $(MAIN_PATH)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "Clean complete"

# Lint the code
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Run the binary (for development)
run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	./$(BUILD_DIR)/$(BINARY_NAME)

# Show help
help:
	@echo "SpectreHub Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build          Build the binary"
	@echo "  release        Build binaries for multiple platforms"
	@echo "  test           Run all tests"
	@echo "  test-contract  Run contract tests"
	@echo "  install        Install to GOPATH/bin"
	@echo "  clean          Remove build artifacts"
	@echo "  lint           Run linter"
	@echo "  fmt            Format code"
	@echo "  tidy           Tidy dependencies"
	@echo "  run            Build and run"
	@echo "  help           Show this help message"
