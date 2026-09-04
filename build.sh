#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="agent"
MAIN_PKG="./cmd/agent"
DIST_DIR="$ROOT_DIR/dist"

usage() {
  cat <<'EOF'
Usage:
  ./build.sh [all|amd64|arm64]

Examples:
  ./build.sh
  ./build.sh all
  ./build.sh amd64
  ./build.sh arm64
EOF
}

target_list() {
  case "${1:-all}" in
    all) printf '%s\n' amd64 arm64 ;;
    amd64|arm64) printf '%s\n' "$1" ;;
    -h|--help|help) usage; exit 0 ;;
    *)
      echo "unknown target: $1" >&2
      usage
      exit 1
      ;;
  esac
}

mkdir -p "$DIST_DIR"

VERSION="${VERSION:-$(git -C "$ROOT_DIR" describe --tags --exact-match 2>/dev/null || git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

for arch in $(target_list "${1:-all}"); do
  out="$DIST_DIR/${APP_NAME}-linux-${arch}"
  echo "building ${out}"
  GOOS=linux GOARCH="$arch" CGO_ENABLED=0 \
    go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
      -o "$out" \
      "$MAIN_PKG"
done

echo "done"
