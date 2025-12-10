# Makefile for terraform-provider-spiceai

default: build

BINARY_NAME=terraform-provider-spiceai
VERSION?=dev
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Directories
PROJECT_DIR=$(shell pwd)
TEST_DIR=$(PROJECT_DIR)/examples/test

.PHONY: all build clean test testacc fmt vet lint install setup plan apply destroy help generate

# Default target
all: build

## Build targets

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

build-all: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)_linux_amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)_linux_arm64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)_darwin_arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)_windows_amd64.exe .

install: build
	@echo "Installing provider to ~/.terraform.d/plugins..."
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/spiceai/spiceai/$(VERSION)/$(GOOS)_$(GOARCH)
	cp $(BINARY_NAME) ~/.terraform.d/plugins/registry.terraform.io/spiceai/spiceai/$(VERSION)/$(GOOS)_$(GOARCH)/

## Test targets

test:
	@echo "Running unit tests..."
	go test ./... -v

testacc:
	@echo "Running acceptance tests..."
	TF_ACC=1 go test ./... -v -timeout 120m

## Code quality targets

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

## Development targets

setup:
	@echo "Setting up ~/.terraformrc with dev_overrides..."
	@printf 'provider_installation {\n  dev_overrides {\n    "spiceai/spiceai" = "%s"\n  }\n  direct {}\n}\n' "$(PROJECT_DIR)" > ~/.terraformrc
	@echo "Done. Provider path: $(PROJECT_DIR)"

plan: build
	@echo "Running terraform plan..."
	@test -n "$$SPICEAI_CLIENT_ID" || (echo "Error: SPICEAI_CLIENT_ID is not set"; exit 1)
	@test -n "$$SPICEAI_CLIENT_SECRET" || (echo "Error: SPICEAI_CLIENT_SECRET is not set"; exit 1)
	cd $(TEST_DIR) && terraform plan

apply: build
	@echo "Running terraform apply..."
	@test -n "$$SPICEAI_CLIENT_ID" || (echo "Error: SPICEAI_CLIENT_ID is not set"; exit 1)
	@test -n "$$SPICEAI_CLIENT_SECRET" || (echo "Error: SPICEAI_CLIENT_SECRET is not set"; exit 1)
	cd $(TEST_DIR) && terraform apply

destroy:
	@echo "Running terraform destroy..."
	@test -n "$$SPICEAI_CLIENT_ID" || (echo "Error: SPICEAI_CLIENT_ID is not set"; exit 1)
	@test -n "$$SPICEAI_CLIENT_SECRET" || (echo "Error: SPICEAI_CLIENT_SECRET is not set"; exit 1)
	cd $(TEST_DIR) && terraform destroy

## Cleanup targets

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)_*
	rm -rf $(TEST_DIR)/.terraform
	rm -f $(TEST_DIR)/.terraform.lock.hcl
	rm -f $(TEST_DIR)/terraform.tfstate
	rm -f $(TEST_DIR)/terraform.tfstate.backup
	@echo "Clean complete"

## Documentation and Code Generation

generate:
	@echo "Running code generation..."
	cd tools && go generate ./...

docs:
	@echo "Generating documentation..."
	tfplugindocs generate --provider-name spiceai

## Dependency management

deps:
	@echo "Downloading dependencies..."
	go mod download

tidy:
	@echo "Tidying dependencies..."
	go mod tidy

## Help

help:
	@echo "Terraform Provider for Spice.ai - Makefile targets"
	@echo ""
	@echo "Build:"
	@echo "  make build       - Build the provider binary"
	@echo "  make build-all   - Build for all platforms"
	@echo "  make install     - Install provider to local plugins directory"
	@echo ""
	@echo "Test:"
	@echo "  make test        - Run unit tests"
	@echo "  make testacc     - Run acceptance tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt         - Format Go code"
	@echo "  make vet         - Run go vet"
	@echo "  make lint        - Run linter"
	@echo ""
	@echo "Code Generation:"
	@echo "  make generate    - Run all code generation (docs, formatting, headers)"
	@echo "  make docs        - Generate provider documentation only"
	@echo ""
	@echo "Development:"
	@echo "  make setup       - Set up ~/.terraformrc with dev_overrides"
	@echo "  make plan        - Build and run terraform plan"
	@echo "  make apply       - Build and run terraform apply"
	@echo "  make destroy     - Run terraform destroy"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean       - Clean build artifacts and state files"
	@echo ""
	@echo "Other:"
	@echo "  make deps        - Download dependencies"
	@echo "  make tidy        - Tidy dependencies"
	@echo ""
	@echo "Environment variables for plan/apply/destroy:"
	@echo "  SPICEAI_CLIENT_ID     - OAuth client ID"
	@echo "  SPICEAI_CLIENT_SECRET - OAuth client secret"
