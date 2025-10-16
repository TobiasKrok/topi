#!/bin/bash
# offline-prep.sh - Prepare for offline development
# Run this BEFORE going offline to pre-pull all required images

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

print_header "Preparing for Offline Development"

# Base images needed for builds
IMAGES=(
    "golang:1.24"
    "debian:12-slim"
    "registry:2"
    "minio/minio:latest"
    "postgres:latest"
    "postgres:14"
    "rabbitmq:3-management"
    "docker.gitea.com/gitea:1.24.3"
)

print_info "Pulling base images..."
for image in "${IMAGES[@]}"; do
    echo "Pulling $image..."
    docker pull "$image"
done
print_success "All base images pulled"

# Build local images
print_header "Building local images"

print_info "Building builder image..."
docker build -t topi-builder:latest -f builder/Dockerfile .
print_success "Builder image built"

print_info "Building scheduler image..."
docker build -t topi-scheduler:latest -f scheduler/Dockerfile .
print_success "Scheduler image built"

# Save Docker Compose images
print_info "Starting docker-compose to cache images..."
docker-compose pull
print_success "Docker Compose images cached"

print_header "Offline Preparation Complete!"
echo ""
echo -e "${GREEN}You're ready for offline development!${NC}"
echo ""
echo "Next steps:"
echo "  1. Run './local-dev-cycle.sh setup' to create your Kind cluster"
echo "  2. Make changes to your code"
echo "  3. Run './local-dev-cycle.sh rebuild' to test"
