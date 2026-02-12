.PHONY: all build clean test install lint fmt vet run help release docker

# Variables
BINARY_NAME := srake
SERVER_BINARY := srake-server
MAIN_PATH := ./cmd/srake
SERVER_PATH := ./cmd/server
WEB_DIR := web
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"
# Enable FTS5 support for SQLite and search features
TAGS := -tags "sqlite_fts5,search"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Default target
all: test build

## help: Display this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'

## build: Build the CLI binary
build:
	@echo "Building $(BINARY_NAME) with FTS5 support..."
	$(GOBUILD) $(TAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: ./$(BINARY_NAME)"

## build-server: Build the server binary
build-server:
	@echo "Building $(SERVER_BINARY) with FTS5 support..."
	$(GOBUILD) $(TAGS) $(LDFLAGS) -o $(SERVER_BINARY) $(SERVER_PATH)
	@echo "Build complete: ./$(SERVER_BINARY)"

## build-all: Build both CLI and server
build-all: build build-server

## build-web: Build the web frontend
build-web:
	@echo "Building web frontend..."
	cd $(WEB_DIR) && npm install && npm run build
	@echo "Web build complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf dist/
	rm -f srake-*.tar.gz srake-*.zip
	@echo "Clean complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) $(TAGS) -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) $(TAGS) -bench=. -benchmem ./internal/processor

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) with FTS5 support..."
	$(GOCMD) install $(TAGS) $(MAIN_PATH)
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"

## tidy: Tidy and verify dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied"

## run: Run the application
run: build
	./$(BINARY_NAME)

## server: Run the API server
server: build-server
	./$(SERVER_BINARY) --port 8080

## web-dev: Run web frontend in development mode
web-dev:
	cd $(WEB_DIR) && npm run dev

## dev-all: Run both server and web frontend in development
dev-all:
	@echo "Starting development environment..."
	@trap 'kill %1' INT; \
	(cd $(WEB_DIR) && npm run dev) & \
	./$(SERVER_BINARY) --port 8080

## run-download: Run download with auto mode
run-download: build
	./$(BINARY_NAME) download --auto

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) .
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

## docker-webapp: Build Docker webapp image
docker-webapp:
	@echo "Building Docker webapp image..."
	docker build -f Dockerfile.webapp -t $(BINARY_NAME)-webapp:$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) .
	@echo "Docker webapp image built: $(BINARY_NAME)-webapp:$(VERSION)"

## docker-compose-up: Start webapp with docker-compose
docker-compose-up:
	@echo "Starting webapp with docker-compose..."
	VERSION=$(VERSION) COMMIT=$(COMMIT) BUILD_DATE=$(BUILD_DATE) \
		docker-compose up --build -d
	@echo "Webapp running at http://localhost:8080"

## docker-compose-down: Stop webapp
docker-compose-down:
	@echo "Stopping webapp..."
	docker-compose down
	@echo "Webapp stopped"

## docker-run: Run Docker container
docker-run: docker
	docker run --rm -v $(PWD)/data:/data $(BINARY_NAME):$(VERSION) --help

## release: Build release binaries
release:
	@echo "Building release binaries..."
	@./scripts/build-release.sh $(VERSION)

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## ci: Run CI pipeline locally
ci: deps check build
	@echo "CI pipeline complete!"

## dev: Start development mode with file watching
dev:
	@echo "Starting development mode..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Install with:"; \
		echo "  go install github.com/cosmtrek/air@latest"; \
		echo ""; \
		echo "Running without file watching..."; \
		$(MAKE) run-server; \
	fi

## stats: Show code statistics
stats:
	@echo "Code statistics:"
	@echo "Lines of code:"
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
	@echo ""
	@echo "Number of files:"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l
	@echo ""
	@echo "Package count:"
	@go list ./... | wc -l

## update: Update all dependencies
update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "Dependencies updated"

## security: Run security checks
security:
	@echo "Running security checks..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with:"; \
		echo "  go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@if command -v nancy > /dev/null; then \
		go list -json -m all | nancy sleuth; \
	else \
		echo "nancy not installed. Install with:"; \
		echo "  go install github.com/sonatype-nexus-community/nancy@latest"; \
	fi

## init: Initialize project for development
init: deps
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/sonatype-nexus-community/nancy@latest
	go install github.com/cosmtrek/air@latest
	@echo "Development environment ready!"

.DEFAULT_GOAL := help