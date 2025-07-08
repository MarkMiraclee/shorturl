# URL Shortener Service

A modern URL shortening service built with Go using clean architecture principles. Converts long URLs to short links with support for multiple storage backends and REST API.

## Features

- URL shortening with unique short IDs
- Multiple storage backends (PostgreSQL, file, in-memory)
- REST API with JSON and text formats
- Batch URL shortening
- User authentication with HMAC-signed cookies
- Gzip compression for requests/responses
- Structured logging with configurable levels
- Health check endpoint

## Tech Stack

- **Language**: Go 1.24.1
- **Router**: Chi v5
- **Database**: PostgreSQL (optional)
- **Logging**: Zap
- **UUID**: Google UUID
- **Architecture**: Clean Architecture

## Getting Started

### Prerequisites
- Go 1.24.1+
- PostgreSQL (optional)

### Installation

```bash
git clone <your-repo-url>
cd shorturl
go mod download
```

### Running

```bash
# Basic (in-memory storage)
go run cmd/shortener/main.go

# With file storage
go run cmd/shortener/main.go -f /path/to/storage.json

# With PostgreSQL
export DATABASE_DSN="postgres://username:password@localhost:5432/dbname?sslmode=disable"
go run cmd/shortener/main.go
```

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_ADDRESS` | HTTP server address | `localhost:8080` |
| `BASE_URL` | Base URL for short links | `http://localhost:8080` |
| `DATABASE_DSN` | PostgreSQL connection string | - |
| `FILE_STORAGE_PATH` | File storage path | - |

### API Examples

```bash
# Shorten URL (text)
curl -X POST http://localhost:8080/ \
  -H "Content-Type: text/plain" \
  -d "https://example.com/very-long-url"

# Shorten URL (JSON)
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very-long-url"}'

# Get original URL
curl http://localhost:8080/abc123

# Health check
curl http://localhost:8080/ping
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/handlers
```
