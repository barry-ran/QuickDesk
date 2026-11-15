#!/usr/bin/env bash
# Build quickdesk-skill-host and its bundled skill MCP servers.
# Run inside the qd-qt-builder image (workspace at /ws) or any Rust toolchain.
set -euo pipefail

WS="${WS:-/ws}"
cd "$WS/quickdesk-skill-host"
cargo build --release --workspace

echo "Built:"
ls -lh target/release/quickdesk-skill-host \
       target/release/sys-info \
       target/release/file-ops \
       target/release/shell-runner
