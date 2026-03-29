# Stage 1 — build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency files first — Docker caches this layer
# so go mod download only reruns when go.mod changes
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# Build the binary
RUN go build -o server ./cmd/api/main.go

# Stage 2 — minimal runtime image
# alpine is only ~5MB vs golang:1.22 which is ~800MB
FROM alpine:latest

WORKDIR /app

# Copy only the compiled binary from stage 1
COPY --from=builder /app/server .

# Expose the port Railway will use
EXPOSE 8080

# Run the binary
CMD ["./server"]