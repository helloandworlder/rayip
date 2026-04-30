#!/usr/bin/env bash
set -euo pipefail

API_URL="${RAYIP_API_URL:-}"
NODE_ID="${RAYIP_NODE_ID:-}"
PROTOCOL="${RAYIP_PROTOCOL:-SOCKS5}"
PORT="${RAYIP_PROBE_PORT:-18080}"
USER_NAME="${RAYIP_SMOKE_USER:-smoke}"
PASSWORD="${RAYIP_SMOKE_PASS:-smoke-pass}"

if [[ -z "$API_URL" ]]; then
  echo "RAYIP_API_URL is required" >&2
  exit 2
fi

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }
}
need_cmd curl
need_cmd node

json_get() {
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(0,'utf8')); const path=process.argv[1].split('.'); let cur=data; for (const key of path) cur=cur?.[key]; if (cur === undefined || cur === null) process.exit(1); console.log(cur)" "$1"
}

curl_json() {
  curl -fsS -H 'content-type: application/json' "$@"
}

curl -fsS "$API_URL/health" >/dev/null
curl -fsS "$API_URL/ready" >/dev/null

nodes_json="$(curl -fsS "$API_URL/api/admin/nodes")"
if [[ -z "$NODE_ID" ]]; then
  NODE_ID="$(printf '%s' "$nodes_json" | json_get 'items.0.id' || true)"
fi
if [[ -z "$NODE_ID" ]]; then
  echo "no connected node found; install NodeAgent first" >&2
  exit 3
fi

account_payload="$(cat <<JSON
{
  "node_id": "$NODE_ID",
  "protocol": "$PROTOCOL",
  "listen_ip": "0.0.0.0",
  "port": $PORT,
  "username": "$USER_NAME",
  "password": "$PASSWORD",
  "egress_limit_bps": 0,
  "ingress_limit_bps": 0,
  "max_connections": 1,
  "desired_generation": 1
}
JSON
)"

created="$(curl_json -X POST "$API_URL/api/admin/runtime-lab/accounts" --data "$account_payload")"
account_id="$(printf '%s' "$created" | json_get 'account.proxy_account_id')"
status="$(printf '%s' "$created" | json_get 'result.status')"
if [[ "$status" != "ACK" && "$status" != "DUPLICATE" ]]; then
  echo "$created" >&2
  echo "runtime apply did not ACK" >&2
  exit 4
fi

curl -fsS "$API_URL/api/admin/runtime-lab/accounts/$account_id/usage" >/dev/null
curl -fsS "$API_URL/api/admin/runtime-lab/nodes/$NODE_ID/digest" >/dev/null
curl_json -X POST "$API_URL/api/admin/nodes/$NODE_ID/scan" >/dev/null

echo "node_id=$NODE_ID"
echo "account_id=$account_id"
echo "protocol=$PROTOCOL"
echo "port=$PORT"
