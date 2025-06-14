# Build stage
FROM golang:1.21-alpine AS builder

# Install git and other build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o main cmd/main.go

# Production stage - minimal Alpine with Python and FFmpeg
FROM alpine:latest

# Install system dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    python3 \
    py3-pip \
    git \
    build-base \
    python3-dev \
    libffi-dev \
    && rm -rf /var/cache/apk/*

# Install Python dependencies (using --break-system-packages since we're in a controlled container environment)
RUN pip3 install --no-cache-dir --break-system-packages --upgrade pip \
    && pip3 install --no-cache-dir --break-system-packages scdl \
    && rm -rf /root/.cache/pip

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Create output directory with proper permissions
RUN mkdir -p output && \
    chown -R appuser:appgroup /app

# Copy the static binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/config ./config

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8000

# Create volume for output files
VOLUME ["/app/output"]

# Verify installations
RUN ffmpeg -version > /dev/null && \
    scdl --version > /dev/null && \
    echo "All dependencies installed successfully"

# Run the application
CMD ["./main", "--port", "8000"] 