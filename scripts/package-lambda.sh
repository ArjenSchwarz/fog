#!/usr/bin/env bash
# Build an AL2023 Lambda archive without depending on the host libc.
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-$repo_root/dist/lambda.zip}"
architecture="${2:-x86_64}"
case "$architecture" in
  x86_64) go_arch=amd64 ;;
  arm64) go_arch=arm64 ;;
  *) echo "Unsupported Lambda architecture: $architecture (expected x86_64 or arm64)" >&2; exit 2 ;;
esac
mkdir -p "$(dirname "$output")"
output="$(cd "$(dirname "$output")" && pwd)/$(basename "$output")"
build_dir="$(mktemp -d)"
trap 'rm -rf "$build_dir"' EXIT
cd "$repo_root"
CGO_ENABLED=0 GOOS=linux GOARCH="$go_arch" go build -trimpath -tags lambda.norpc -ldflags "-s -w" -o "$build_dir/bootstrap" .
chmod 755 "$build_dir/bootstrap"
# Always create a fresh archive: zip otherwise retains obsolete entries.
(cd "$build_dir" && zip -q -X lambda.zip bootstrap)
mv "$build_dir/lambda.zip" "$output"
