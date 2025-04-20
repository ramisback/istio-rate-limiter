# Variables
PROTO_DIR = user-service/proto
GO = go
DOCKER = docker
KUBECTL = kubectl
ISTIOCTL = istioctl
MODULE_PATH = github.com/ramisback/istio-rate-limiter

# Colors for output
GREEN = \033[0;32m
NC = \033[0m # No Color

.PHONY: all build run clean proto docker-build k8s-deploy k8s-delete test lint help fix-modules

# Default target
all: build

# Build all services
build:
	@echo "$(GREEN)Building user service...$(NC)"
	cd user-service && $(GO) build -o bin/user-service
	@echo "$(GREEN)Building rate limit service...$(NC)"
	cd rate-limit-service && $(GO) build -o bin/rate-limit-service

# Run services locally
run:
	@echo "$(GREEN)Starting user service...$(NC)"
	cd user-service && $(GO) run main.go
	@echo "$(GREEN)Starting rate limit service...$(NC)"
	cd rate-limit-service && $(GO) run main.go

# Generate protobuf files
proto:
	@echo "$(GREEN)Generating protobuf files...$(NC)"
	protoc --go_out=. --go_opt=module=$(MODULE_PATH) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE_PATH) \
		$(PROTO_DIR)/*.proto

# Fix module paths in proto files
fix-modules:
	@echo "$(GREEN)Fixing module paths in proto files...$(NC)"
	sed -i '' 's|github.com/ram/istio-rate-limiter|github.com/ramisback/istio-rate-limiter|g' $(PROTO_DIR)/*.proto

# Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	rm -rf user-service/bin
	rm -rf rate-limit-service/bin
	rm -f $(PROTO_DIR)/*.pb.go

# Build Docker images
docker-build:
	@echo "$(GREEN)Building Docker images...$(NC)"
	$(DOCKER) build -t user-service:latest ./user-service
	$(DOCKER) build -t rate-limit-service:latest ./rate-limit-service

# Deploy to Kubernetes
k8s-deploy:
	@echo "$(GREEN)Deploying to Kubernetes...$(NC)"
	$(KUBECTL) apply -f k8s/
	@echo "$(GREEN)Waiting for deployments to be ready...$(NC)"
	$(KUBECTL) wait --for=condition=available --timeout=300s deployment/user-service
	$(KUBECTL) wait --for=condition=available --timeout=300s deployment/ratelimit

# Delete from Kubernetes
k8s-delete:
	@echo "$(GREEN)Deleting from Kubernetes...$(NC)"
	$(KUBECTL) delete -f k8s/

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	cd user-service && $(GO) test ./...
	cd rate-limit-service && $(GO) test ./...

# Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run ./...

# Install dependencies
deps:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Run load tests
loadtest:
	@echo "$(GREEN)Running load tests...$(NC)"
	cd loadtest && $(GO) run main.go

# Help command
help:
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  make build        - Build all services"
	@echo "  make run          - Run services locally"
	@echo "  make proto        - Generate protobuf files"
	@echo "  make fix-modules  - Fix module paths in proto files"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker images"
	@echo "  make k8s-deploy   - Deploy to Kubernetes"
	@echo "  make k8s-delete   - Delete from Kubernetes"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linter"
	@echo "  make deps         - Install dependencies"
	@echo "  make loadtest     - Run load tests"
	@echo "  make help         - Show this help message" 