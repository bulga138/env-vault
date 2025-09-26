# syntax=docker/dockerfile:1

# ---- Stage 1: Builder ----
# Use a specific version of the golang:alpine image for reproducibility.
FROM golang:1.22-alpine AS builder

# Set the working directory.
WORKDIR /app

# Install necessary build tools.
# git is needed to pull dependencies, ca-certificates is for HTTPS calls during build.
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum first to leverage Docker layer caching.
# This layer only rebuilds if your dependencies change.
COPY go.mod go.sum ./

# Download dependencies.
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Build the application as a static binary.
# CGO_ENABLED=0 creates a binary that doesn't depend on C libraries.
# -ldflags="-s -w" strips debug information, making the binary smaller.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /env-vault .


# ---- Stage 2: Final Image ----
# Use the 'scratch' image, which is the smallest possible base. It's completely empty.
FROM scratch

# Set the working directory.
WORKDIR /

# The scratch image is empty, so we need to copy essential files.
# Copy the compiled binary from the builder stage.
COPY --from=builder /env-vault /env-vault

# Copy the SSL certificate authorities from the builder stage.
# This is crucial for your app to make secure HTTPS requests (like to GitHub's API).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Although this app doesn't expose a port, this line is kept as a good practice example.
# EXPOSE 8080

# Set the entrypoint for the container.
# This makes the container act like the binary itself.
ENTRYPOINT ["/env-vault"]