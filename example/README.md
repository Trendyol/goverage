# Example Application

This example demonstrates how to integrate `goverage` into a Go application.

## Requirements

- Go 1.20 or higher

## Quick Start

1. **Build the application:**
   ```bash
   ./build.sh
   ```

2. **Run the application:**
   ```bash
   export GOCOVERDIR=/tmp/coverage-example
   ./example-app
   ```

3. **Test the endpoints:**
   ```bash
   # Health check
   curl http://localhost:8080/api/health
   
   # Calculator
   curl "http://localhost:8080/api/calculate?a=5&b=3"
   ```

4. **Get coverage data:**
   ```bash
   curl -X POST http://localhost:7777/v1/cover/profile
   ```

## Automated Test

Run the complete test workflow:

```bash
./test.sh
```

This script will:
- Build the app with coverage
- Start the server
- Exercise the endpoints
- Collect coverage data
- Display the results

## Understanding the Coverage

The example application includes several code paths:
- Basic HTTP handlers
- Conditional logic in the calculator
- Different calculation scenarios (positive, zero, negative numbers)

By testing different endpoints and parameters, you can see how coverage changes in real-time.

## Build Comparison

**Without coverage:**
```bash
go build -o example-app .
# Coverage server is NOT included (due to build tags)
```

**With coverage:**
```bash
go build -cover -covermode=atomic -tags=goverage -o example-app .
# Coverage server IS included and starts automatically
```
