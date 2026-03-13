.PHONY: build install clean test

BINARY_NAME=skillctl
BUILD_DIR=build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .

install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/ 2>/dev/null || \
		echo "Please copy $(BUILD_DIR)/$(BINARY_NAME) to a directory in your PATH"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test -v ./...

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

run:
	@go run main.go