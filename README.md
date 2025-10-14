# goverage

A Go library that provides runtime code coverage collection via HTTP API. It is a goc-compatible alternative that implements the same `/v1/cover/profile` endpoint using Go's official `covdata` tool, making it easy to pull runtime coverage from long-running services while keeping production builds clean. See the original goc project: [qiniu/goc](https://github.com/qiniu/goc).

## Features

- 🔄 **Runtime Coverage**: Collect coverage data from running applications
- 📊 **Standard Format**: Returns coverage data in Go's standard textfmt format
- 🏷️ **Build Tags**: Only included when built with `goverage` tag
- 🌐 **HTTP API**: Compatible with goc's `/v1/cover/profile` endpoint
- ⚡ **Atomic Mode**: Supports real-time coverage data collection

## Installation

> **Requirements**: Go 1.20 or higher

Add the library to your Go project:

```bash
go get github.com/Trendyol/goverage
```

## Usage

### 1. Import the Library

Add the import to your main application (or any Go file that gets imported) as a blank import:

```go
package main

import (
    _ "github.com/Trendyol/goverage" // Import with blank identifier
    // ... other imports
)

func main() {
    // Your application code
    // Coverage server will automatically start in background
}
```

### 2. Build with Coverage

Build your application with coverage and the `coverage` tag:

```bash
go build -cover -covermode=atomic -tags=goverage -o ./your-app .
```

**Important**: Use `-covermode=atomic` for real-time coverage data collection! Once the app starts, goverage listens on port `7777` by default and serves coverage via `POST /v1/cover/profile`.

### 3. Runtime Requirements

Your runtime environment needs the `covdata` executable (Go's official coverage data tool). The recommended approach is conditional copying based on build mode:

```dockerfile
# Build stage - conditionally copy covdata
RUN if [ "$ARG_MODE" = "automation_test" ]; then go tool covdata -h >/dev/null 2>&1 || :; cp "$(go tool -n covdata | awk '{print $1}')" "/app/libs/"; fi


# Runtime stage - copy to /usr/bin/
COPY --from=prod-build /app/libs/ /usr/bin/
```

This ensures the `covdata` tool is only included when needed for coverage testing.

### 4. Environment Variables

Configure the coverage server using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `COVERAGE_HTTP_PORT` | HTTP server port | `7777` |
| `GOCOVERDIR` | Directory to write coverage data | **Required** |

### 5. Running the Application

#### For Development/Local Testing:
```bash
export GOCOVERDIR=/tmp/coverage
mkdir -p $GOCOVERDIR
./your-app
```

#### For Production/Automated Testing with Docker:
Use the automation-run.sh script pattern:

```bash
#!/bin/sh
env GOCOVERDIR="/tmp/coverage/" /app/your-app
```

#### Build Arguments for Docker:
When building with Docker, use the `ARG_MODE` build argument:

```bash
# For production build (no coverage)
docker build -t your-app .

# For coverage testing
docker build -tags=goverage --build-arg ARG_MODE=coverage -t your-app-coverage .
```

The coverage server will start automatically and listen on the configured port when built with coverage support.

## API Usage

### Get Coverage Profile

**Endpoint**: `POST /v1/cover/profile`

**Request Body** (optional):
```json
{
    "skipFile": ["pattern1", "pattern2"]
}
```

**Response**: Coverage data in Go's textfmt format

**Example**:
```bash
# Get all coverage data from default port 7777
curl -X POST http://localhost:7777/v1/cover/profile

# Skip certain files using regex patterns
curl -X POST http://localhost:7777/v1/cover/profile \
  -H "Content-Type: application/json" \
  -d '{"skipFile": [".*test.*", ".*mock.*"]}'
```

**Response Format**:
```
mode: atomic
your/package/file.go:10.2,12.16 2 1
your/package/file.go:12.16,14.3 1 0
your/package/other.go:20.1,22.2 1 1
```

## Docker Integration

Here's a production-ready Dockerfile example with conditional coverage support and explicit `covdata` copy for coverage builds:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

ARG ARG_MODE
ARG BUILD_FLAGS=""

WORKDIR /app

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Conditionally copy covdata tool only for coverage builds (required at runtime)
RUN mkdir -p /app/libs && \
    if [ "$ARG_MODE" = "automation_test" ]; then \
        go tool covdata -h >/dev/null 2>&1 || :; cp "$(go tool -n covdata | awk '{print $1}')" "/app/libs/"; \
    fi

# Build with coverage support using build script
RUN /bin/sh .deploy/build.sh

# Runtime stage  
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy application and scripts
COPY --from=builder /app/your-app /app/your-app
COPY --from=builder /app/.deploy/run.sh /app/run.sh

# Copy covdata tool (only present in coverage builds)
COPY --from=builder /app/libs/ /usr/bin/

# Expose both application and coverage server ports
EXPOSE 8080 7777

ENTRYPOINT ["/bin/sh", "/app/run.sh"]
```

### Build Scripts

Create `.deploy/build.sh` for conditional building (`coverage` tag enables goverage):

```bash
#!/bin/sh

if [ "$ARG_MODE" = "automation_test" ]
then
   go build -cover -covermode=atomic -tags=goverage -o ./your-app .
else
   go build -ldflags="-w -s" -o ./your-app .
fi
```

Create `.deploy/run.sh` for runtime:

```bash
#!/bin/sh
env GOCOVERDIR="/tmp/coverage/" /app/your-app
```

## Build Tags Explained

The library uses Go build tags to ensure it's only included in coverage builds:

- **Production builds**: `go build -o app .` → Coverage server **not included**
- **Coverage builds**: `go build -tags=goverage -cover -o app .` → Coverage server **included**

This ensures zero overhead in production deployments.


## Troubleshooting

### Common Issues

1. **"GOCOVERDIR is not set" error**
   ```bash
   export GOCOVERDIR=/tmp/coverage
   mkdir -p $GOCOVERDIR
   ```

2. **"covdata not found" error**
   - Ensure `covdata` binary is available in runtime environment
   - Check the binary path in your Docker image

3. **Empty coverage data**
   - Ensure you're using `-covermode=atomic` for real-time data
   - Verify your application is being exercised (receiving requests)

4. **Port conflicts**
   ```bash
   export COVERAGE_HTTP_PORT=8888
   ```

### Logs

The coverage server provides detailed logging:
```
coverage server listening on :7777
incoming request method=POST path=/v1/cover/profile remote=127.0.0.1:54321
served textfmt bytes=1234 duration=45ms
```

## Compatibility

- **Go Version**: 1.19+
- **Coverage Mode**: atomic
- **Platforms**: Linux, macOS, Windows
- **Architecture**: amd64, arm64

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Example

See the [example directory](./example/) for a complete working example that demonstrates:
- How to integrate the library into a Go application
- Build scripts with proper flags
- Automated testing workflow
- Coverage data collection

**Key Implementation Details:**
- **Build Mode:** Conditional building based on `ARG_MODE=coverage`
- **Docker Integration:** Multi-stage build with conditional covdata tool copying
- **Runtime Environment:** `/tmp/coverage/` directory with proper permissions
- **Port Configuration:** Application on `:8080`, Coverage server on `:7777`

**Benefits Achieved:**
- Zero-overhead production builds (coverage code excluded)
- Real-time coverage collection during automated testing
- Seamless integration with existing CI/CD pipelines
- No application code changes required

Production implementations validate the library's reliability and ease of integration across different environments.

## Acknowledgments

Inspired by the [goc](https://github.com/qiniu/goc) project for comprehensive Go coverage testing.
