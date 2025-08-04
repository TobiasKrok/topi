# Build the engine binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum* go.sum

# Download dependencies
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o engine cmd/engine/main.go

# Use distroless as minimal base image to package the engine binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/engine .
USER 65532:65532

ENTRYPOINT ["/engine"]