# Build stage
FROM golang:1.23-alpine AS builder

# Install git for build info
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum (if they exist)
COPY go.* ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build arguments for version information
ARG GIT_COMMIT
ARG VERSION

# Build the application with version information
RUN go build -v -ldflags "-X 'utils.Version=${VERSION}' -X 'utils.Commit=${GIT_COMMIT}'" -o /app/tailbone

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/tailbone /app/tailbone

# Run the binary
ENTRYPOINT ["/app/tailbone"] 