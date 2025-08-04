# Build the scheduler binary
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
COPY api/ api/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o scheduler cmd/scheduler/main.go

# Use distroless as minimal base image to package the scheduler binary
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/scheduler .
USER 65532:65532

ENTRYPOINT ["/scheduler"]