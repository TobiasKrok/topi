# Makefile for topi project

# Build configuration
BUILD_DIR = out
MAIN_APP = main
BINARIES = engine

.PHONY: all build run clean test watch docker-run docker-down itest help

# Default target
all: test build

# Build all binaries
build:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@for %%b in ($(BINARIES)) do @(echo Building %%b... && go build -o $(BUILD_DIR)\%%b.exe .\cmd\%%b)

# Run a specific binary
run: build
	@if "$(filter-out $@,$(MAKECMDGOALS))" == "" ( \
		$(BUILD_DIR)\$(MAIN_APP).exe \
	) else ( \
		$(BUILD_DIR)\$(filter-out $@,$(MAKECMDGOALS)).exe \
	)

# Allow any target as an argument to 'run'
%:
	@:

# Clean up binary from last build
clean:
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)

# Run tests
test:
	@go test ./... -v

# Run integration tests
itest:
	@go test ./... -tags=integration -v

# Live reload the application (requires air)
watch:
	@air

# Create DB container
docker-run:
	@docker-compose up -d

# Shutdown DB container
docker-down:
	@docker-compose down

# Install dependencies
install:
	@go mod tidy
	@go mod download

# List available binaries
list-binaries:
	@echo Available binaries:
	@echo - engine (main application)
	@dir /b .\cmd 2>nul

# Help information
help:
	@echo Makefile commands:
	@echo   make all         - Run tests and build application
	@echo   make build       - Build the application
	@echo   make run         - Run the main application
	@echo   make run <name>  - Run a specific binary (e.g., make run engine)
	@echo   make clean       - Clean up binaries
	@echo   make test        - Run tests
	@echo   make itest       - Run integration tests
	@echo   make watch       - Live reload with air
	@echo   make docker-run  - Create DB container
	@echo   make docker-down - Shutdown DB container
	@echo   make install     - Install dependencies
	@echo   make list-binaries - List available binaries
	@echo   make help        - Show this help message