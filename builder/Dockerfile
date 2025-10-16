# Build the builder binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy shared module first (it's a dependency)
COPY shared/ shared/

# Copy the Go Modules manifests
COPY builder/go.mod builder/go.mod
COPY builder/go.sum* builder/go.sum

WORKDIR /workspace/builder

# Download dependencies
RUN go mod download

# Copy the go source
COPY builder/cmd/ cmd/
COPY builder/internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o builder cmd/main.go

# Use Debian slim for better compatibility with installation scripts
FROM debian:12-slim

# Install essential tools and create workspace
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    git \
    ca-certificates \
    curl \
    wget \
    gnupg \
    lsb-release \
    sudo \
    build-essential \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /opt/topi/workspace /opt/topi/system \
    && chmod 777 /opt/topi/workspace \
    && chmod 755 /opt/topi/system

WORKDIR /opt/topi/workspace
COPY --from=builder /workspace/builder/builder /usr/local/bin/builder

# Run as root to allow package installations in workflows
# For production, use non-root with specific capabilities
USER root
ENTRYPOINT ["/usr/local/bin/builder"]
