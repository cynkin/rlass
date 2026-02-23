# Stage 1: Build the Go binary
# We use a full Go image just to compile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files first (Docker caches this layer) Copy into app/
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile to a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o rlaas .

# Stage 2: Run with a minimal image
# We don't need Go installed to RUN a compiled binary
FROM alpine:3.19

WORKDIR /app

# Copy only the compiled binary from Stage 1
COPY --from=builder /app/rlaas .

# Copy migrations folder
COPY --from=builder /app/migrations ./migrations

# Expose gRPC and admin API ports
EXPOSE 50051 8090

# go run .
CMD ["./rlaas"]