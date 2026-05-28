FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy source
COPY main.go go.mod go.sum ./

# Build with minimal attack surface
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /runner main.go

# Minimal runtime image — no shell, no package manager
FROM scratch

COPY --from=builder /runner /runner

# Run as non-root
USER 65534:65534

ENTRYPOINT ["/runner"]
