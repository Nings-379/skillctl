.PHONY: build install clean test dist

BINARY_NAME=skillctl
BUILD_DIR=build
DIST_DIR=dist

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .

dist:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/skillctl-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o $(DIST_DIR)/skillctl-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/skillctl-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/skillctl-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/skillctl-darwin-arm64 .
	@echo "Done. Binaries in $(DIST_DIR)/"

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