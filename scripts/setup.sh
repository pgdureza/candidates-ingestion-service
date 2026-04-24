#!/bin/bash
set -e

echo "=== Candidate Ingestion Service Setup ==="

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Install Go 1.23+"
    exit 1
fi
echo "✓ Go $(go version)"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Install Docker"
    exit 1
fi
echo "✓ Docker installed"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose not found. Install Docker Compose"
    exit 1
fi
echo "✓ Docker Compose installed"

# Setup Go modules
echo ""
echo "Setting up Go modules..."
go mod download
go mod tidy
echo "✓ Go modules ready"

# Create .env if not exists
if [ ! -f .env ]; then
    cp .env.example .env
    echo "✓ Created .env from template (customize if needed)"
fi

# Build Docker image
echo ""
echo "Building Docker image..."
docker build -t candidate-ingestion:latest .
echo "✓ Docker image built"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Start services: make up"
echo "  2. In another terminal: make run (or continue with docker-compose)"
echo "  3. Test: curl http://localhost:8080/health"
echo "  4. Send webhook: see README.md for examples"
echo ""
echo "For Kubernetes:"
echo "  1. kubectl cluster-info"
echo "  2. make k8s-deploy"
echo "  3. kubectl get pods"
