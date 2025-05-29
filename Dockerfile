# Stage 1: Build the Go application
FROM docker.io/golang:1.21-alpine AS builder

WORKDIR /app

# Copy the Go source file
# If you had go.mod and go.sum, you would copy them first and download dependencies
# COPY go.mod go.sum ./
# RUN go mod download
COPY test.go .

# Build the application
# Output will be named 'appmain' (or any name you choose)
RUN go build -o /app/appmain test.go

# Stage 2: Create the final lightweight image
FROM docker.io/alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/appmain .

# Expose the port the server listens on (if running in server mode)
EXPOSE 8989

# Set the entrypoint for the container.
# The user will need to specify 'server' or 'client' as a command when running the container.
# e.g., docker run myapp server
# or docker run -e SERVER_ADDR=host.docker.internal:8989 -e SEND_PERIOD=2s myapp client
ENTRYPOINT ["/app/appmain"]
