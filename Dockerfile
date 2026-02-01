# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Set Go proxy for faster downloads in China
ENV GOPROXY=https://goproxy.cn,direct

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server/

# Production stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
	CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
ENTRYPOINT ["./server"]
