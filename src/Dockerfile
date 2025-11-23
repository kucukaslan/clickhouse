# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Install git for build info and swag for API documentation
RUN apk add --no-cache git

# Install swag CLI for generating Swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy go mod files
COPY go.mod go.sum ./

RUN go mod download

# Copy all source code
COPY . ./

# Generate Swagger documentation
RUN swag init -g main.go --output ./docs

# Build the application with ldflags to inject build information
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-X kucukaslan/clickhouse/buildinfo.Version=$(git describe --tags --always 2>/dev/null || echo 'dev') \
              -X kucukaslan/clickhouse/buildinfo.Commit=$(git rev-parse HEAD 2>/dev/null || echo 'unknown') \
              -X kucukaslan/clickhouse/buildinfo.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o main .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 50051

CMD ["./main"]
