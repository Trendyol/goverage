#!/bin/bash

# Example build script showing how to build with coverage support

echo "Building example application with coverage support..."

# Set coverage directory
export GOCOVERDIR=/tmp/coverage-example
mkdir -p $GOCOVERDIR

# Build with coverage and goverage tag
echo "Building with coverage..."
go build -cover -covermode=atomic -tags=goverage -o example-app .

echo "Build completed!"
echo ""
echo "To run:"
echo "  export GOCOVERDIR=/tmp/coverage-example"
echo "  ./example-app"
echo ""
echo "To get coverage:"
echo "  curl -X POST http://localhost:7777/v1/cover/profile"
