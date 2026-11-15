#!/usr/bin/env bash
# Configure + build the QuickDesk Qt client on Linux.
# Runs natively on a host with Qt >= 6.5, or inside the qt-builder image:
#   docker build -t qd-qt-builder -f scripts/linux/qt-builder.Dockerfile scripts/linux
#   docker run --rm -v "$PWD":/ws -w /ws --user "$(id -u):$(id -g)" \
#     -e HOME=/tmp qd-qt-builder bash scripts/build_qd_linux.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

BUILD_TYPE="${BUILD_TYPE:-Release}"

cmake -S . -B output/linux-build -G Ninja \
    -DCMAKE_BUILD_TYPE="$BUILD_TYPE" "$@"
cmake --build output/linux-build -j"$(nproc)"

# MCP bridge, packaged next to QuickDesk (MainController::getMcpBinaryPath).
if command -v cargo >/dev/null 2>&1; then
    cargo build --release --locked --manifest-path quickdesk-mcp/Cargo.toml
else
    echo "cargo not found: quickdesk-mcp not built (package_appimage.sh will skip it)" >&2
fi