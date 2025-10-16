#!/bin/bash

set -e

CONTAINER_NAME="ship-status-test-db"
DB_PORT="5433"
DB_USER="postgres"
DB_PASSWORD="testpass"
DB_NAME="ship_status_test"

cleanup() {
  echo "Cleaning up dashboard processes..."
  pkill -f "go run.*cmd/dashboard" 2>/dev/null || true
  sleep 1
  
  echo "Cleaning up test container..."
  podman stop $CONTAINER_NAME 2>/dev/null || true
  podman rm $CONTAINER_NAME 2>/dev/null || true
}

trap cleanup EXIT

echo "Cleaning up any existing test container..."
podman stop $CONTAINER_NAME 2>/dev/null || true
podman rm $CONTAINER_NAME 2>/dev/null || true

echo "Starting PostgreSQL container..."
podman run -d \
  --name $CONTAINER_NAME \
  -e POSTGRES_PASSWORD=$DB_PASSWORD \
  -p $DB_PORT:5432 \
  quay.io/enterprisedb/postgresql:latest

echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
  if podman exec $CONTAINER_NAME pg_isready -U $DB_USER > /dev/null 2>&1; then
    echo "PostgreSQL is ready"
    break
  fi
  if [ $i -eq 30 ]; then
    echo "PostgreSQL failed to start"
    podman logs $CONTAINER_NAME
    podman stop $CONTAINER_NAME
    podman rm $CONTAINER_NAME
    exit 1
  fi
  sleep 1
done

echo "Creating test database..."
podman exec $CONTAINER_NAME psql -U $DB_USER -c "CREATE DATABASE $DB_NAME;"

DSN="postgres://$DB_USER:$DB_PASSWORD@localhost:$DB_PORT/$DB_NAME?sslmode=disable&client_encoding=UTF8"
export TEST_DATABASE_DSN="$DSN"

echo "Running e2e tests..."
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT/test/e2e"
set +e
gotestsum --format testname -- -timeout 30s .
TEST_EXIT_CODE=$?
set -e

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo "✓ Tests passed"
else
  echo "✗ Tests failed with exit code $TEST_EXIT_CODE"
fi

exit $TEST_EXIT_CODE

