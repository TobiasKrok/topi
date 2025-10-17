# Build the scheduler binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy shared module first (it's a dependency)
COPY shared/ shared/

# Copy the Go Modules manifests
COPY scheduler/go.mod scheduler/go.mod
COPY scheduler/go.sum scheduler/go.sum

WORKDIR /workspace/scheduler

# Download dependencies
RUN go mod download

# Copy the go source
COPY scheduler/cmd/ cmd/
COPY scheduler/api/ api/
COPY scheduler/internal/ internal/
COPY scheduler/config/ config/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o scheduler cmd/main.go

# Use distroless as minimal base image to package the scheduler binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/scheduler/scheduler .
USER 65532:65532

ENTRYPOINT ["/scheduler"]