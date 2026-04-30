#!/usr/bin/env bash
set -euo pipefail

OWNER_REPO="${RAYIP_GITHUB_REPO:-helloandworlder/rayip}"
INSTALL_ROOT="${RAYIP_INSTALL_ROOT:-/opt/rayip}"
ENV_DIR="${RAYIP_ENV_DIR:-/etc/rayip}"
SERVICE_NAME="${RAYIP_SERVICE_NAME:-rayip-node-agent}"
COMPONENT="${1:-}"
shift || true

if [[ "$COMPONENT" != "node-agent" ]]; then
  echo "usage: rayip-install.sh node-agent \"-t <token> -u <api-url> [--grpc-url <grpc-url>] [--node-code <code>] [--version <tag>] [--port <port>]\"" >&2
  exit 2
fi

if [[ $# -eq 1 ]]; then
  # Supports nyanpass-style quoted argument string after the component name.
  eval "set -- $1"
fi

token=""
api_url=""
grpc_url=""
node_code="$(hostname)"
version="latest"
probe_port="18080"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -t|--token) token="${2:-}"; shift 2 ;;
    -u|--url) api_url="${2:-}"; shift 2 ;;
    --grpc-url) grpc_url="${2:-}"; shift 2 ;;
    --node-code) node_code="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    --port) probe_port="${2:-}"; shift 2 ;;
    -h|--help)
      sed -n '1,40p' "$0"
      exit 0
      ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$token" || -z "$api_url" ]]; then
  echo "-t/--token and -u/--url are required" >&2
  exit 2
fi

if [[ -z "$grpc_url" ]]; then
  host="$(printf '%s' "$api_url" | sed -E 's#^https?://##; s#/.*$##')"
  grpc_url="grpcs://${host}:443"
fi

if [[ "$version" == "latest" ]]; then
  base_url="https://github.com/${OWNER_REPO}/releases/latest/download"
  release_dir="$INSTALL_ROOT/releases/latest"
else
  base_url="https://github.com/${OWNER_REPO}/releases/download/${version}"
  release_dir="$INSTALL_ROOT/releases/${version}"
fi

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }
}
need_cmd curl
need_cmd shasum
need_cmd systemctl

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

download() {
  local name="$1"
  curl -fL --retry 3 --retry-delay 2 -o "$tmp_dir/$name" "$base_url/$name"
}

download rayip-node-agent-linux-amd64
download rayip-xray-linux-amd64
download runtime-manifest.json
download SHA256SUMS

(
  cd "$tmp_dir"
  shasum -a 256 -c SHA256SUMS
)

install -d -m 0755 "$release_dir" "$INSTALL_ROOT/runtime" "$ENV_DIR"
install -m 0755 "$tmp_dir/rayip-node-agent-linux-amd64" "$release_dir/rayip-node-agent-linux-amd64"
install -m 0755 "$tmp_dir/rayip-xray-linux-amd64" "$release_dir/rayip-xray-linux-amd64"
install -m 0644 "$tmp_dir/runtime-manifest.json" "$release_dir/runtime-manifest.json"
install -m 0644 "$tmp_dir/runtime-manifest.json" "$INSTALL_ROOT/runtime/runtime-manifest.json"

ln -sfn "$release_dir" "$INSTALL_ROOT/current"

cat > "$ENV_DIR/node-agent.env" <<EOF
RAYIP_AGENT_NODE_CODE=$node_code
RAYIP_AGENT_ENROLLMENT_TOKEN=$token
RAYIP_AGENT_API_GRPC_ADDR=$grpc_url
RAYIP_AGENT_RUNTIME_BUNDLE_DIR=$INSTALL_ROOT/runtime
RAYIP_AGENT_RUNTIME_CORE_MODE=xray
RAYIP_AGENT_RUNTIME_XRAY_GRPC_ADDR=auto
RAYIP_AGENT_RUNTIME_XRAY_BINARY_PATH=$INSTALL_ROOT/current/rayip-xray-linux-amd64
RAYIP_AGENT_RUNTIME_XRAY_AUTO_START=true
RAYIP_AGENT_PROBE_PORT=$probe_port
RAYIP_AGENT_PROBE_PROTOCOLS=SOCKS5,HTTP
RAYIP_AGENT_LEASE_INTERVAL=10s
RAYIP_AGENT_LEASE_TTL=45s
EOF

cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=RayIP NodeAgent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_DIR/node-agent.env
ExecStart=$INSTALL_ROOT/current/rayip-node-agent-linux-amd64
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

echo "node_code=$node_code"
systemctl --no-pager --full status "$SERVICE_NAME" || true
echo "logs: journalctl -u $SERVICE_NAME -f"
