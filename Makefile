.PHONY: build install clean test run help

BINARY_NAME=shelly-gitops
INSTALL_PATH=/usr/local/bin

help:
	@echo "Shelly Git-Ops Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build      - Build the binary"
	@echo "  make install    - Install to /usr/local/bin"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run tests"
	@echo "  make run        - Run from source"
	@echo "  make tidy       - Tidy go modules"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/shelly-gitops

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete!"

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

test:
	@echo "Running tests..."
	go test -v ./...

run:
	@echo "Running from source..."
	go run ./cmd/shelly-gitops

tidy:
	@echo "Tidying go modules..."
	go mod tidy

deps:
	@echo "Downloading dependencies..."
	go mod download
