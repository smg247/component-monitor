#!/bin/bash

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "Building dashboard image..."
podman build -f images/dashboard/Dockerfile -t ship-status-dashboard .

echo "Stopping existing container (if any)..."
podman rm -f ship-status-dashboard 2>/dev/null || true

echo "Starting dashboard container..."
podman run -d \
  --name ship-status-dashboard \
  -p 8090:8080 \
  -v "$PROJECT_ROOT/deploy/api/config.yaml:/app/config.yaml:ro" \
  ship-status-dashboard \
  --config /app/config.yaml \
  --port 8080 \
  --dsn "postgres://postgres:postgres@host.containers.internal:5432/ship_status?sslmode=disable&client_encoding=UTF8"

echo ""
echo "✓ Dashboard is running on http://localhost:8090"
echo "Test endpoints:"
echo "  curl http://localhost:8090/health"
echo "  curl http://localhost:8090/api/components"

