# Runtime Release, ZTP, and Smoke

## Release Assets

Tag `v*` triggers `.github/workflows/release.yml` and publishes:

- `rayip-node-agent-linux-amd64`
- `rayip-xray-linux-amd64`
- `runtime-manifest.json`
- `rayip-install.sh`
- `SHA256SUMS`

Local dry run:

```bash
make release-dry-run
```

## ZTP Install

```bash
bash <(curl -fLSs https://github.com/<owner>/<repo>/releases/latest/download/rayip-install.sh) node-agent "-t <enrollment-token> -u <api-url>"
```

Optional flags:

- `--grpc-url <host:port>` overrides the gRPC endpoint derived from `-u`.
- `--node-code <code>` defaults to `hostname`.
- `--version <tag>` installs a fixed release instead of latest.
- `--port <port>` sets the probe port, default `18080`.

The installer downloads release artifacts only, verifies `SHA256SUMS`, writes `/etc/rayip/node-agent.env`, atomically switches `/opt/rayip/current`, installs `rayip-node-agent.service`, and restarts systemd.

## Runtime Node Smoke

After a real Linux node is online:

```bash
RAYIP_API_URL=https://<railway-api-domain> \
RAYIP_PROTOCOL=SOCKS5 \
RAYIP_PROBE_PORT=18080 \
bash scripts/smoke-runtime-node.sh
```

The smoke script verifies `/health`, `/ready`, picks the first connected node unless `RAYIP_NODE_ID` is set, creates a Runtime Lab account, requires RuntimeApply ACK, then queries usage, digest, and node scan.

## Scan Reason Codes

Current platform reason codes:

- `no_candidate_public_ip`
- `private_ip`
- `cgnat`
- `ingress_unreachable`
- `auth_failed`
- `unsupported_protocol`
- `egress_mismatch`
- `scan_timeout`
- `digest_mismatch`

The first pass implements stable platform storage/response semantics and reachability classification. Protocol auth, egress mismatch, timeout specialization, and digest mismatch hooks are reserved for the full external scanner.
