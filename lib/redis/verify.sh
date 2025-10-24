#!/bin/bash

echo "=== Redis Client Verification ==="
echo ""

echo "1. Checking Go files..."
go list -f '{{.GoFiles}}' github.com/coder/agentapi/lib/redis
echo ""

echo "2. Checking test files..."
go list -f '{{.TestGoFiles}}' github.com/coder/agentapi/lib/redis
echo ""

echo "3. Running go vet..."
go vet ./client.go ./health.go ./doc.go
if [ $? -eq 0 ]; then
    echo "✓ go vet passed"
else
    echo "✗ go vet failed"
fi
echo ""

echo "4. Running go fmt check..."
gofmt -l client.go health.go doc.go
echo "✓ go fmt check complete"
echo ""

echo "5. Checking package documentation..."
go doc -short .
echo ""

echo "6. File statistics..."
echo "client.go: $(wc -l < client.go) lines"
echo "client_test.go: $(wc -l < client_test.go) lines"
echo "health.go: $(wc -l < health.go) lines"
echo ""

echo "=== Verification Complete ==="
