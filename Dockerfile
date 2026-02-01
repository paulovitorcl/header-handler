# Build stage
FROM golang:1.22-alpine AS build

# Install git for fetching dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o controller \
    ./cmd/manager

# Final stage
FROM gcr.io/distroless/base-debian12

LABEL org.opencontainers.image.source="https://github.com/seu-user/header-route-controller"
LABEL org.opencontainers.image.description="Kubernetes controller for header-based routing with Envoy"
LABEL org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /

# Copy the binary from build stage
COPY --from=build /app/controller /controller

# Use non-root user for security
USER nonroot:nonroot

# Expose health and metrics ports
EXPOSE 8081 8082

ENTRYPOINT ["/controller"]
