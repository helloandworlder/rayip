#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
VERSION="${RAYIP_RELEASE_VERSION:-${GITHUB_REF_NAME:-dev}}"
XRAY_VERSION="${RAYIP_XRAY_VERSION:-$VERSION}"
BUILD_ID="${RAYIP_BUILD_ID:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

mkdir -p "$DIST_DIR"
rm -f \
  "$DIST_DIR/rayip-node-agent-linux-amd64" \
  "$DIST_DIR/rayip-xray-linux-amd64" \
  "$DIST_DIR/runtime-manifest.json" \
  "$DIST_DIR/rayip-install.sh" \
  "$DIST_DIR/SHA256SUMS"

cd "$ROOT_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$DIST_DIR/rayip-node-agent-linux-amd64" ./services/node-agent/cmd/node-agent

cd "$ROOT_DIR/third_party/xray-core"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$DIST_DIR/rayip-xray-linux-amd64" ./main

cd "$ROOT_DIR"
cp deploy/node-agent/rayip-install.sh "$DIST_DIR/rayip-install.sh"
chmod 0755 "$DIST_DIR/rayip-install.sh"

agent_sha="$(shasum -a 256 "$DIST_DIR/rayip-node-agent-linux-amd64" | awk '{print $1}')"
xray_sha="$(shasum -a 256 "$DIST_DIR/rayip-xray-linux-amd64" | awk '{print $1}')"
install_sha="$(shasum -a 256 "$DIST_DIR/rayip-install.sh" | awk '{print $1}')"

tmp_manifest="$(mktemp)"
cat > "$tmp_manifest" <<JSON
{
  "bundle_version": "$VERSION",
  "xray_version": "$XRAY_VERSION",
  "extension_abi": "rayip.runtime.v1",
  "build_id": "$BUILD_ID",
  "binaries": {
    "node_agent_linux_amd64": {
      "asset": "rayip-node-agent-linux-amd64",
      "sha256": "$agent_sha"
    },
    "xray_linux_amd64": {
      "asset": "rayip-xray-linux-amd64",
      "sha256": "$xray_sha"
    },
    "installer": {
      "asset": "rayip-install.sh",
      "sha256": "$install_sha"
    }
  },
  "binary_sha256": "sha256:$agent_sha",
  "manifest_sha256": "",
  "signature": "",
  "capabilities": [
    "socks5",
    "http",
    "mixed",
    "rayip-runtime",
    "account-rate-limit",
    "smart-fair-limit",
    "congestion-aware-fair-limit",
    "connection-limit",
    "usage-stats",
    "abuse-detection",
    "runtime-digest"
  ]
}
JSON

manifest_sha="$(shasum -a 256 "$tmp_manifest" | awk '{print $1}')"
sed "s/\"manifest_sha256\": \"\"/\"manifest_sha256\": \"sha256:$manifest_sha\"/" "$tmp_manifest" > "$DIST_DIR/runtime-manifest.json"
rm -f "$tmp_manifest"

(
  cd "$DIST_DIR"
  shasum -a 256 \
    rayip-node-agent-linux-amd64 \
    rayip-xray-linux-amd64 \
    runtime-manifest.json \
    rayip-install.sh > SHA256SUMS
)

echo "release artifacts written to $DIST_DIR"
