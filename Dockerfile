# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o oas-mcp ./cmd/oas-mcp

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/oas-mcp .

# Copy default config files
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/swagger.json .

# Expose port for HTTP mode
EXPOSE 8080

# Default command
ENTRYPOINT ["./oas-mcp"]
CMD ["--config=config.yaml"] 