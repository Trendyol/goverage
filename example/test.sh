#!/bin/bash

# Example test script showing coverage collection workflow

echo "Coverage Collection Test Script"
echo "==============================="

# Setup
export GOCOVERDIR=/tmp/coverage-example
mkdir -p $GOCOVERDIR

echo "1. Building application with coverage..."
go build -cover -covermode=atomic -tags=goverage -o example-app .

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "2. Starting application..."
./example-app &
APP_PID=$!

# Wait for app to start
sleep 2

echo "3. Testing application endpoints..."
curl -s http://localhost:8080/api/health > /dev/null
curl -s "http://localhost:8080/api/calculate?a=5&b=3" > /dev/null
curl -s "http://localhost:8080/api/calculate?a=0&b=10" > /dev/null
curl -s "http://localhost:8080/api/calculate?a=-2&b=-3" > /dev/null

echo "4. Collecting coverage data..."
curl -X POST http://localhost:7777/v1/cover/profile -o coverage.out

if [ -f coverage.out ]; then
    echo "5. Coverage data collected successfully!"
    echo "Coverage report:"
    echo "----------------"
    cat coverage.out
    echo ""
    echo "To view HTML report: go tool cover -html=coverage.out -o coverage.html"
else
    echo "5. Failed to collect coverage data"
fi

echo "6. Cleaning up..."
kill $APP_PID
rm -f example-app coverage.out

echo "Test completed!"
