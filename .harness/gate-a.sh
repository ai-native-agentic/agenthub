#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

echo "[Gate A] Format + Vet check for agenthub (Go)"

# gofmt check
echo "  Running gofmt..."
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor || true)
if [ -n "$UNFORMATTED" ]; then
    echo "[FAIL] Unformatted files:"
    echo "$UNFORMATTED"
    exit 1
fi
echo "  [PASS] gofmt clean"

# go vet
echo "  Running go vet..."
go vet ./... 2>&1
echo "  [PASS] go vet clean"

echo "[Gate A] PASSED"
