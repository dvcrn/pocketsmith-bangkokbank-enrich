FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o bkkbank-enrich

FROM alpine:latest

RUN apk add --no-cache tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bkkbank-enrich .

# Run the application
ENTRYPOINT ["./bkkbank-enrich"]
