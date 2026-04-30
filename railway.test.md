# RayIP Railway Test Deployment

目标项目：`RayIP`
目标环境：`Test`

## Services

- `Postgres`
- `Redis`
- `nats`
- `api`

`nats` command:

```bash
nats-server -js -sd /data
```

Mount a Railway volume at `/data` for JetStream file storage.

## API Variables

```bash
RAYIP_SERVICE_ENV=test
RAYIP_HTTP_ADDR=:8080
RAYIP_GRPC_ADDR=:9090
RAYIP_POSTGRES_DSN=${{Postgres.DATABASE_URL}}
RAYIP_POSTGRES_RUN_MIGRATIONS=true
RAYIP_REDIS_ADDR=${{Redis.REDIS_PRIVATE_URL}}
RAYIP_NATS_URL=nats://nats.railway.internal:4222
RAYIP_NODE_ENROLLMENT_TOKEN=<rotate-before-node-install>
```

Deploy from this repo root:

```bash
railway up --service api --environment Test
```

The API image uses `services/api/Dockerfile` and exposes HTTP `8080` plus gRPC `9090`.

## Health Gates

```bash
curl -fsS "$RAYIP_API_URL/health"
curl -fsS "$RAYIP_API_URL/ready"
```

After the first NodeAgent connects, verify:

```bash
curl -fsS "$RAYIP_API_URL/api/admin/nodes"
curl -fsS "$RAYIP_API_URL/api/admin/runtime-lab/nodes/<node-id>/digest"
```
