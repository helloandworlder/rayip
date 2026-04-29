# GoSea-Light 迁移 + CI/CD 生产化方案

> 版本：v2 · 2026-04-24
>
> **双目标**：
> 1. 把自运营生产 `light.gosea.in` 从过载的 Dokploy 宿主机，迁到隔离的 Railway 项目 `GoSeaLight-Self`
> 2. 把 **LiuNian-GoSeaLight**（客户租户）和 **GoSeaLight-Self**（自运营租户）两个**互不相关的 prod** 统一到同一套 tag-drive CI/CD：一条 `git tag v*` → 测试 gate → build GHCR → 两个 Railway 项目同时 redeploy
>
> **范围澄清**：两个 Railway 项目数据**完全独立**（独立 DB / 域名 / env / 用户 / 订单）。仅共用代码仓库和镜像。**本方案不处理跨项目数据同步**（那是另一个话题）。

## 1. 背景 & 决策

**为什么迁**：2 核 7.5G Dokploy 宿主机挤了 37 个容器，load 28、CPU idle 0，`/api/orders/my/storefront-zones` 超 30s。救火清掉 18 个非关键容器后已恢复到 ~100ms，但共用宿主机风险永远存在（随时可能因为同机上新增项目再过载）。Railway 单项目隔离资源，`GoSeaLight-Self` 独占 CPU/RAM。

**为什么统一 CI/CD**：
- LiuNian 目前 `source.repo=null`，每次更新要手工 `railway up --path-as-root GoSea-Light/backend`（memory 有记录"Railway 落后 Dokploy 4 天"的事故）
- Dokploy 当前：tag push → Actions build → GHCR → webhook 触发 redeploy（已有）
- 目标：两个 Railway 项目走相同姿势——监听 GHCR 镜像 tag，Actions 在测试通过后触发两者同时 redeploy。单发版源，不再人肉 `railway up`

**不做的事**：
- 不趁机"顺便"加缓存 / 改代码 / 升级镜像版本 —— 迁移就是迁移，代码变化留给下次 tag
- 不做跨租户数据同步（两个 prod 是独立业务）
- 不保留 Dokploy 作为热备 —— 源主机以后做"只读冷备"1~2 周后下线

## 2. 现状清点（源 = Dokploy on 43.160.244.246）

| 项目 | 值 |
|---|---|
| 源域名 | `light.gosea.in`（Cloudflare DNS，Traefik + LE） |
| Dokploy 应用 slug | `gosealight-app-fkvf0t` |
| 容器 | `gosealight-frontend` / `gosealight-backend` / `gosealight-postgres` / `gosealight-redis` |
| 卷 | `gosealight-app-fkvf0t_platform_{backend_storage,postgres_data,redis_data}` |
| DB | `xray_platform`，83 MB；pg 用户 `postgres`/`20030928`；端口 5432（容器网） |
| 最大表 | `zone_inventory_snapshots` 44 MB / 28 行（bloat，迁后 `VACUUM FULL`） |
| 业务表行数 | `order_purchases` / `user_node_ip_leases` / `audit_logs` 均 0 行（新库） |
| Backend 镜像 | GHCR `ghcr.io/helloandworlder/gosea-light-backend:vX.Y.Z` |
| Frontend 镜像 | GHCR `ghcr.io/helloandworlder/gosea-light-frontend:vX.Y.Z` |
| 部署触发 | GitHub tag `v*` push（见 `.github/workflows/docker-publish.yml`） |
| 必须同步的 env | `APP_MASTER_KEY` · `JWT_SECRET` · `JWT_EXPIRES_IN` · `DEFAULT_ADMIN_USERNAME` · `DEFAULT_ADMIN_PASSWORD` · `NODE_ENV=production` · `PORT=3000` · `PRISMA_BASELINE_ON_P3005=false` · `PRISMA_BASELINE_SKIP_LATEST=true` · `SNOWFLAKE_MACHINE_ID=1` |
| Railway 自动注入 | `DATABASE_URL`（`${{Postgres.DATABASE_URL}}`） · `REDIS_URL`（`${{Redis.REDIS_URL}}`） |

⚠️ **关键加密 key**：`APP_MASTER_KEY` 在 `backend/src/common/crypto/credential-crypto.service.ts:15` 用来 AES 加解密节点凭据。**不同步会导致所有已存节点的密码/token 无法解密**，节点全部失联。

⚠️ **JWT_SECRET 同步**：不同步，现有登录 token 全失效（用户要重新登录，管理员也是）。

## 3. Railway 侧初始化（T-24h 做）

### 3.1 创建项目 + 服务

```bash
cd /Users/yuxi/SynexIM-Project/Xray-Tool
railway login                                      # 确认登录 helloandworlder@outlook.com
railway init --name GoSeaLight-Self                # 创建新项目
railway link                                       # 选刚建的 GoSeaLight-Self

# 加插件服务
railway add --database postgres
railway add --database redis

# 建应用服务（空壳）
railway service create backend
railway service create frontend
```

### 3.2 部署方式（唯一选择：GHCR 镜像 + 滚动 channel tag）

Railway dashboard → backend service → Settings → Source → **Image** → 填：
```
ghcr.io/helloandworlder/gosea-light-backend:stable
```
frontend 同理 `...-frontend:stable`。

**为什么不用源码 `railway up`**：
- LiuNian 以前就是这个姿势，导致和 Dokploy 漂移（memory 记录 "Railway 跑 4 天前的代码"）
- CI/CD 统一发车的前提是"唯一的 immutable artifact" = GHCR 镜像
- 源码部署无法做到 Actions 测试绿了再部署，等于绕过 gate

`:stable` 是由 GitHub Actions 在 **tag 发版 + 测试全绿** 的时刻推送的滚动标签（见 §3.5）。Railway Deploy Hook 被 Actions 触发后拉最新 `:stable`。

原始 `.railwayignore` 里 `archive/` 和 `xraytool/` 的排除规则保留（万一以后还需要 `railway up` 临时应急）。

### 3.3 环境变量同步

从源抓 env（已做过）：
```bash
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  'docker exec gosealight-backend printenv' > /tmp/src-env.txt
```

写入 Railway（**不要**把 DATABASE_URL / REDIS_URL 手工写死）：
```bash
railway service backend
railway variables \
  --set "APP_MASTER_KEY=<源值>" \
  --set "JWT_SECRET=<源值>" \
  --set "JWT_EXPIRES_IN=168h" \
  --set "DEFAULT_ADMIN_USERNAME=byx" \
  --set "DEFAULT_ADMIN_PASSWORD=<源值>" \
  --set "NODE_ENV=production" \
  --set "PORT=3000" \
  --set "PRISMA_BASELINE_ON_P3005=false" \
  --set "PRISMA_BASELINE_SKIP_LATEST=true" \
  --set "SNOWFLAKE_MACHINE_ID=1" \
  --set 'DATABASE_URL=${{Postgres.DATABASE_URL}}' \
  --set 'REDIS_URL=${{Redis.REDIS_URL}}'
```

frontend 只需 `BACKEND_UPSTREAM=http://backend.railway.internal:3000`（见 `frontend/nginx.conf.template`）。

### 3.4 卷挂载

Dokploy backend 容器目前挂了 `platform_backend_storage` 卷（用途：备份文件 / 可能的上传产物）。确认挂载路径：
```bash
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  'docker inspect gosealight-backend | grep -A5 Mounts'
```

Railway backend service → Settings → Volumes → Add：mount 到相同路径（大概率是 `/app/storage` 或 `/app/data`，先看源 inspect）。10G 配额起。

### 3.5 CI/CD 改造（新增，最重要的一节）

**触发路径**：
```
git tag -a vX.Y.Z -m "release: ..."
git push github vX.Y.Z
        ↓
   ┌─ ci-tests job (必须绿才进下一步)
   │   ├── backend: lint + prisma validate + jest + build
   │   └── frontend: typecheck + build
        ↓
   ├─ build-images job (needs: ci-tests)
   │   ├── buildx build & push ghcr.io/.../gosea-light-backend:vX.Y.Z + :stable
   │   └── buildx build & push ghcr.io/.../gosea-light-frontend:vX.Y.Z + :stable
        ↓
   └─ deploy job (needs: build-images, matrix: [liunian, self])
       ├── curl POST $RAILWAY_LIUNIAN_BACKEND_HOOK
       ├── curl POST $RAILWAY_LIUNIAN_FRONTEND_HOOK
       ├── curl POST $RAILWAY_SELF_BACKEND_HOOK
       ├── curl POST $RAILWAY_SELF_FRONTEND_HOOK
       └── curl POST $DOKPLOY_WEBHOOK_URL  (保留过渡，Self 迁完可删)
```

**`.github/workflows/ci.yml` 改造**（每次 push/PR 都跑）：

```yaml
name: CI
on:
  push:
    branches: ["**"]
  pull_request:

jobs:
  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: xray_platform
        options: >-
          --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5
        ports: ["5432:5432"]
    defaults:
      run: { working-directory: backend }
    env:
      DATABASE_URL: postgresql://postgres:postgres@127.0.0.1:5432/xray_platform?schema=public
      APP_MASTER_KEY: ci_dummy_master_key
      JWT_SECRET: ci_dummy_jwt_secret
      NODE_ENV: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20, cache: pnpm }
      - uses: pnpm/action-setup@v4
        with: { version: 10 }
      - run: pnpm install --frozen-lockfile
      - run: pnpm db:generate
      - run: pnpm exec prisma validate       # 新增
      - run: pnpm lint                       # 新增
      - run: pnpm test                       # 新增：11 个 spec
      - run: pnpm build

  frontend:
    runs-on: ubuntu-latest
    defaults:
      run: { working-directory: frontend }
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v2
        with: { bun-version: 1.3.8 }
      - run: bun install --frozen-lockfile
      - run: bun run build   # vue-tsc + vite build 已包含 typecheck
```

**`.github/workflows/release.yml`（替换原 docker-publish.yml）**：

```yaml
name: Release
concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false
on:
  push:
    tags: ["v*"]
  workflow_dispatch:

permissions:
  contents: read
  packages: write

jobs:
  ci-tests:
    uses: ./.github/workflows/ci.yml   # reusable，保证 tag 发版前测试绿

  build-images:
    needs: ci-tests
    runs-on: ubuntu-latest
    outputs:
      tag: ${{ steps.meta.outputs.tag }}
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - id: meta
        run: echo "tag=${GITHUB_REF_NAME}" >> $GITHUB_OUTPUT
      - uses: docker/build-push-action@v6
        with:
          context: ./backend
          push: true
          tags: |
            ghcr.io/${{ github.repository_owner }}/gosea-light-backend:${{ steps.meta.outputs.tag }}
            ghcr.io/${{ github.repository_owner }}/gosea-light-backend:stable
      - uses: docker/build-push-action@v6
        with:
          context: ./frontend
          push: true
          tags: |
            ghcr.io/${{ github.repository_owner }}/gosea-light-frontend:${{ steps.meta.outputs.tag }}
            ghcr.io/${{ github.repository_owner }}/gosea-light-frontend:stable

  deploy:
    needs: build-images
    runs-on: ubuntu-latest
    strategy:
      matrix:
        target:
          - { name: liunian-backend,  hook: RAILWAY_LIUNIAN_BACKEND_HOOK }
          - { name: liunian-frontend, hook: RAILWAY_LIUNIAN_FRONTEND_HOOK }
          - { name: self-backend,     hook: RAILWAY_SELF_BACKEND_HOOK }
          - { name: self-frontend,    hook: RAILWAY_SELF_FRONTEND_HOOK }
    steps:
      - name: Trigger Railway redeploy - ${{ matrix.target.name }}
        env:
          HOOK_URL: ${{ secrets[matrix.target.hook] }}
        run: |
          if [ -z "$HOOK_URL" ]; then
            echo "::warning::${{ matrix.target.name }} hook not configured"
            exit 0
          fi
          curl --fail --show-error --silent --retry 5 --retry-delay 5 \
               -X POST "$HOOK_URL"

  deploy-dokploy:
    needs: build-images
    runs-on: ubuntu-latest
    # 迁移过渡期保留；Self 彻底迁到 Railway 后删
    if: ${{ vars.DOKPLOY_ENABLED == 'true' }}
    steps:
      - name: Trigger Dokploy
        env:
          DOKPLOY_WEBHOOK_URL: ${{ secrets.DOKPLOY_WEBHOOK_URL }}
        run: |
          [ -n "$DOKPLOY_WEBHOOK_URL" ] || exit 0
          curl --fail --show-error --silent --retry 5 --retry-delay 5 \
               -X POST "$DOKPLOY_WEBHOOK_URL" \
               -H "Content-Type: application/json" \
               -d "{\"ref\":\"$GITHUB_REF_NAME\",\"source\":\"github-actions\"}"
```

**branch protection**（GitHub repo settings → Branches → main）：
- Require pull request before merging
- Require status checks: `backend` + `frontend` 必须绿
- 推 tag 不受 branch protection 影响，但 tag 来源 commit 若在 main 上，就经过了测试 gate

### 3.6 Railway Deploy Hook 配置

**两个项目各自在 dashboard 操作**：

LiuNian-GoSeaLight（现有，迁移 + 改造）：
1. backend service → Settings → Source → Image → 改 `ghcr.io/helloandworlder/gosea-light-backend:stable`
2. backend service → Settings → Deploy Triggers → Generate Deploy Webhook → 复制 URL
3. frontend service → 同上两步
4. 删除原 `source.repo=null` 的上传产物（Railway 会在下次 deploy 后清理）

GoSeaLight-Self（新建）：
1. §3.1 建完项目后，每个 service 都按上面两步配 Image + Webhook

**GitHub repo secrets**（helloandworlder/GoSea-Light → Settings → Secrets and variables → Actions）：
```
RAILWAY_LIUNIAN_BACKEND_HOOK   = https://backboard.railway.app/...
RAILWAY_LIUNIAN_FRONTEND_HOOK  = https://backboard.railway.app/...
RAILWAY_SELF_BACKEND_HOOK      = https://backboard.railway.app/...
RAILWAY_SELF_FRONTEND_HOOK     = https://backboard.railway.app/...
DOKPLOY_WEBHOOK_URL            = （保留原值，过渡期使用）
```

**GitHub variables**（非 secret）：
```
DOKPLOY_ENABLED = true（过渡期）→ false（迁完后）
```

### 3.7 LiuNian 数据保留（⚠️ 最关键）

LiuNian 当前是**真生产数据**（见 baseline），改造 CI/CD **绝对不能碰 Postgres service**。

**基线（2026-04-24 抓的，改造前最后一次记录）**：

| 指标 | 值 |
|---|---|
| DB 总大小 | 130 MB |
| `order_purchases` | 4,241 行 |
| `order_purchase_items` | 4,241 行 |
| `user_node_ip_leases` | 4,241 行 |
| `wallet_ledger` | 336 行 |
| `audit_logs` | 197 行 |
| `user_accounts` | 12 行 |
| `zones` | 2 行 |
| `zone_node_bindings` | 10 行 |
| `zone_inventory_snapshots` | 48,360 行（88 MB bloat） |

**保护措施**：
1. **改造前强制冷备到本地**（必须，不可跳过）：
   ```bash
   railway service Postgres
   DBU=$(railway variables --kv | grep DATABASE_PUBLIC_URL= | cut -d= -f2-)
   docker run --rm -e PGURL="$DBU" postgres:16-alpine \
     sh -c 'pg_dump "$PGURL" -Fc -Z 6 --no-owner --no-acl' \
     > ~/backup/liunian-pre-cicd-$(date +%Y%m%d-%H%M).dump
   ls -lh ~/backup/liunian-pre-cicd-*.dump   # 预期 10-30 MB
   ```
2. **改造时只改 backend/frontend 两个服务的 Source 字段**：
   - 不 delete 任何 service
   - 不 delete 任何 volume
   - 不改 Postgres / Redis 的 Settings
   - 不改 `DATABASE_URL` 引用（仍然是 `${{Postgres.DATABASE_URL}}`）
3. **改造后校验行数**：
   ```bash
   docker run --rm -e PGURL="$DBU" postgres:16-alpine \
     psql "$PGURL" -c "SELECT relname, n_live_tup FROM pg_stat_user_tables \
       WHERE relname IN ('order_purchases','user_accounts','wallet_ledger','user_node_ip_leases') \
       ORDER BY relname;"
   # 必须与基线一致
   ```

**如果改造过程误操作导致数据损坏**：
```bash
# 用冷备 restore 到一个新 Postgres service 实例（不要覆盖原库，先隔离）
railway service create Postgres-Restore
# 拿 Postgres-Restore 的 PUBLIC URL，pg_restore 进去
# 验证数据正确后，再改 backend 的 DATABASE_URL 引用
```

### 3.8 Railway 原生自动备份（必须配）

**两个 Railway 项目都要配**（LiuNian + Self）：

Dashboard 路径：Postgres service → Settings → **Backups** → Enable。

| 配置 | 推荐 | 说明 |
|---|---|---|
| Backup schedule | Daily | Railway 原生每日 snapshot |
| Retention | 7 days（或 plan 最大值） | 覆盖一周回滚窗口 |
| Restore path | Dashboard One-click | Railway 有 "Restore from backup" 按钮 |

**如果当前 plan 不支持原生备份**（Developer plan 有限制）：必须升级到 **Pro plan**。对有 4241 订单的 LiuNian，升级费用远低于数据丢失成本。

**验证备份已生效**：Dashboard → Postgres → Backups 页能看到至少 1 个 snapshot，带时间戳。

### 3.9 双保险：GitHub Actions 定时 pg_dump 外部备份

Railway 原生备份挂在同一个 Railway 账户下——万一账户被封、计费暂停、区域故障，备份也失效。加一层**异地冷备**：

新增 `.github/workflows/backup.yml`：

```yaml
name: Postgres Backup
on:
  schedule:
    - cron: "17 3 * * *"   # 每天 UTC 03:17（北京 11:17）— 避开整点
  workflow_dispatch:

jobs:
  backup:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        target:
          - { name: liunian, url_secret: RAILWAY_LIUNIAN_DB_URL }
          - { name: self,    url_secret: RAILWAY_SELF_DB_URL }
    steps:
      - name: Install postgres-client
        run: sudo apt-get update && sudo apt-get install -y postgresql-client-16
      - name: Dump
        env:
          DB_URL: ${{ secrets[matrix.target.url_secret] }}
        run: |
          [ -n "$DB_URL" ] || { echo "::warning::${{ matrix.target.name }} DB URL not set"; exit 0; }
          TS=$(date -u +%Y%m%d-%H%M)
          pg_dump "$DB_URL" -Fc -Z 9 --no-owner --no-acl \
            -f ${{ matrix.target.name }}-$TS.dump
          ls -lh ${{ matrix.target.name }}-$TS.dump
      - name: Upload to R2 / S3
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.BACKUP_S3_KEY }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.BACKUP_S3_SECRET }}
          AWS_DEFAULT_REGION: auto
          S3_ENDPOINT: ${{ secrets.BACKUP_S3_ENDPOINT }}   # R2 endpoint
          S3_BUCKET: ${{ secrets.BACKUP_S3_BUCKET }}
        run: |
          [ -n "$AWS_ACCESS_KEY_ID" ] || { echo "::warning::no S3 creds"; exit 0; }
          aws s3 cp ${{ matrix.target.name }}-*.dump \
            s3://$S3_BUCKET/gosealight/${{ matrix.target.name }}/ \
            --endpoint-url $S3_ENDPOINT
      - name: Prune > 30 days on S3 (optional)
        run: |
          # R2 lifecycle rule 更好，但也可以在 workflow 里 aws s3 ls + 删
          true
```

**Secrets 需要添加**：
```
RAILWAY_LIUNIAN_DB_URL = <LiuNian DATABASE_PUBLIC_URL 完整值>
RAILWAY_SELF_DB_URL    = <Self DATABASE_PUBLIC_URL 完整值>
BACKUP_S3_KEY / SECRET / ENDPOINT / BUCKET = <Cloudflare R2 / S3 凭据>
```

**选型建议**：Cloudflare R2（免出口流量费，10 GB 免费），完全够用一年的每日全量（130 MB × 365 = 47 GB，压缩后 ~15 GB）。

**恢复演练**：每季度从 R2 下一份 dump，在 Railway 建一个临时 Postgres → restore → 查行数。不做演练的备份等于没有备份。

## 4. 数据迁移（T-0 停机窗口）

### 4.1 停机前预演（T-24h 做一次，不切域名）

```bash
# 源端：导出但不停机，看看 dump 大小和时间
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  "docker exec gosealight-postgres pg_dump -U postgres -d xray_platform -F c -Z 6 --no-owner --no-acl -f /tmp/dry.dump && ls -lh /tmp/dry.dump"
# 预期：<15s，~10-20 MB
```

### 4.2 正式停机窗口（5-15 分钟）

**T+0 冻结源**  
```bash
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  'docker stop gosealight-backend'           # 停 backend，frontend 留着吐静态页
```

**T+1 最终 dump**（custom format，含 `_prisma_migrations` 以规避 P3005）
```bash
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  "docker exec gosealight-postgres pg_dump -U postgres -d xray_platform \
   -F c -Z 6 --no-owner --no-acl -f /tmp/final.dump"

# 拉回本地
sshpass -p '20030928BByx@' scp root@43.160.244.246:/tmp/final.dump /tmp/final.dump
```

**T+2 Restore 到 Railway Postgres**
```bash
# 拿 Railway 外部连接 URL（注意：是 proxy 地址，不是 internal）
railway service Postgres
railway variables | grep DATABASE_PUBLIC_URL   # postgresql://postgres:xxx@roundhouse.proxy.rlwy.net:xxxxx/railway

# Restore（单事务 + 清空目标）
pg_restore --clean --if-exists --no-owner --no-acl \
  --single-transaction \
  -d "postgresql://postgres:xxx@roundhouse.proxy.rlwy.net:xxxxx/railway" \
  /tmp/final.dump
```

**Prisma 兼容性取舍**：因为 dump 里带了 `_prisma_migrations`，Railway backend 重启时 `prisma migrate deploy` 会发现已全部 applied，直接跳过 —— 无需 baseline。`PRISMA_BASELINE_ON_P3005=false` 兜底，万一 P3005 也不会乱 baseline。

**T+3 卷数据迁移**
```bash
# 源端打包（停机时容器已停，卷不会被写）
sshpass -p '20030928BByx@' ssh root@43.160.244.246 \
  "docker run --rm -v gosealight-app-fkvf0t_platform_backend_storage:/src \
   -v /tmp:/dst alpine tar czf /dst/storage.tar.gz -C /src ."
sshpass -p '20030928BByx@' scp root@43.160.244.246:/tmp/storage.tar.gz /tmp/storage.tar.gz

# 注入 Railway volume：临时用 railway run 在 backend service 上下文里 exec
railway service backend
cat /tmp/storage.tar.gz | railway run --service backend -- \
  sh -c 'tar xzf - -C /app/storage'   # 路径以实际挂载为准

# 或者：把 tar 放进镜像构建上下文作为一次性 init，一次启动即解压到 volume
```

> 如果 backend_storage 实际是空的（新库，没产出过备份文件），可以跳过卷迁移。先 `ls -la` 源端卷内容再决定。

**T+4 启动 Railway backend**（通过 Deploy Hook 拉 `:stable`）  
```bash
curl -X POST "$RAILWAY_SELF_BACKEND_HOOK"
curl -X POST "$RAILWAY_SELF_FRONTEND_HOOK"
```
看 Railway dashboard 的 deployment 日志应出现 `Nest application successfully started` 和 `ZonesInventoryCron` 开跑。

**T+5 冒烟验证**（见第 6 节清单）

## 5. 域名切换

### 5.1 T-24h：预降 TTL

Cloudflare → DNS → `light.gosea.in` 这条 CNAME → TTL 改 `Auto` 为 **60 秒**。**保持 "DNS only"（灰云）**，Railway LE 签发需要直连。

### 5.2 T-2h：Railway 端加自定义域名

Railway dashboard → frontend service → Settings → Networking → Custom Domain → 填 `light.gosea.in`。  
Railway 给出 **两条记录**：
- `TXT _railway-challenge.light.gosea.in` 某随机值（证书挑战）
- `CNAME light.gosea.in → xxx.up.railway.app`（流量）

先只加 TXT 到 Cloudflare，**CNAME 不要动**。等 Railway 状态变 `Domain verified`（~1 分钟），LE 证书通常 2-3 分钟签完。

> ⚠️ `railway domain` CLI 历史报 Unauthorized（可能 plan 限制），**一律走 dashboard**。

### 5.3 T-0 停机窗口内：切 CNAME

Cloudflare → 改 `light.gosea.in` 的 CNAME target 为 Railway 给的值 → 保存 → **继续保持灰云**。TTL 60s → 最坏 1-2 分钟全球生效。

### 5.4 观察

```bash
# 本机实时看 DNS 收敛
for i in 1 2 3 4 5; do
  echo "=== 8.8.8.8 ==="; dig +short light.gosea.in @8.8.8.8
  echo "=== 1.1.1.1 ==="; dig +short light.gosea.in @1.1.1.1
  sleep 20
done
```

## 6. 切换后验证清单

| # | 项 | 命令 / 操作 | 预期 |
|---|---|---|---|
| 1 | DNS 指向 | `dig +short light.gosea.in @8.8.8.8` | Railway 的 CNAME 目标 |
| 2 | HTTPS 证书 | `curl -vI https://light.gosea.in 2>&1 \| grep "issuer\|subject"` | issuer = Let's Encrypt |
| 3 | 前端静态页 | `curl -sS https://light.gosea.in \| head -20` | 返回 index.html |
| 4 | API 健康 | `curl -sS -o /dev/null -w "%{http_code} %{time_total}s\n" https://light.gosea.in/api/orders/my/storefront-zones` | 401 + <300ms |
| 5 | 登录流程 | 浏览器登录 `byx` / 原密码 | 成功（证明 JWT_SECRET 同步对） |
| 6 | 下单页加载 | 打开用户下单页，切地区 | 正常出 zones 列表，不超时 |
| 7 | Prisma migrations | Railway backend logs 首次启动 | 出现 "migrate deploy" 且无 P3005 |
| 8 | 节点连通性 | 管理员后台查看节点列表 | 节点状态 reachable（证明 APP_MASTER_KEY 同步对） |
| 9 | Cron | backend 日志 | `ZonesInventoryCron done elapsed=XXms zones=5` 每 2 分钟 |
| 10 | 延迟对比 | 对比救火后 Dokploy (~100ms) 和 Railway | Railway 同级或更好 |

## 7. 回滚方案

触发回滚条件：验证清单任一项失败且 10 分钟内修不好。

```bash
# Cloudflare: 把 light.gosea.in CNAME 改回 Dokploy 原值（home.365proxy.net 的原 Traefik 目标）
# 因为 TTL 60s，1-2 分钟回滚完成

# 源端 backend 重新启动（数据没写过，安全）
sshpass -p '20030928BByx@' ssh root@43.160.244.246 'docker start gosealight-backend'
```

回滚后 Railway 项目保留，分析失败原因后再切。

## 8. 风险矩阵

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| `APP_MASTER_KEY` 漏同步 → 节点全部失联 | 中 | 严重 | 清单 §3.3 显式列出；验证 §6 第 8 条 |
| `JWT_SECRET` 漏同步 → 所有用户被踢 | 中 | 中 | 同上 |
| Cloudflare 橙云导致 LE 签发失败 | 高 | 中 | **明确灰云**（DNS only）；§5.1 |
| Prisma migrations 乱 baseline | 低 | 中 | `PRISMA_BASELINE_ON_P3005=false`；dump 带 `_prisma_migrations` |
| Railway 免费 plan 自定义域名被拒 | 低 | 高 | T-24h 先 dashboard 加域名确认能用；不行立即升 plan |
| 卷路径不一致 | 中 | 低 | T-24h 先 `docker inspect` 确认源挂载路径再配 Railway |
| Railway internal DNS 没连通 | 低 | 中 | backend 起不来会立即在 log 报错，dashboard 检查 Service Networking |
| 源机再次过载影响迁移窗口 | 中 | 低 | 迁移前再 docker stop 一次非关键容器，dump 时保证资源 |
| Deploy Hook URL 泄露 → 任意 POST 触发部署 | 低 | 中 | 放 GitHub secrets；不要 commit 到 repo；定期在 Railway dashboard rotate |
| `:stable` 漂移导致回滚不干净 | 中 | 中 | 回滚时用 Railway dashboard 的 deployment history "Redeploy" 老 deployment（它锁版 image digest，不受 `:stable` 影响） |
| CI 服务只有 Postgres 没 Redis，部分 spec 可能 skip | 低 | 低 | 若某 spec 依赖 Redis，再加 `redis:7-alpine` service（见 §3.5 模板预留扩展） |
| tag 绕过 main 直接从 feature branch 打 | 中 | 中 | 发版 checklist：只在 main 最新 commit 打 tag；reusable ci-tests 仍会跑但前置 commit 可能未 merge |
| LiuNian 改造时误删 Postgres service 导致 4241 订单丢失 | 低（操作层面） | **致命** | §3.7 冷备强制；改造 checklist 明确"只改 backend/frontend Source，不碰 Postgres/Redis"；§3.8 Railway 原生备份 + §3.9 R2 异地备份双保险 |
| Railway 账户被封 / 区域故障 → 原生备份同归于尽 | 低 | **致命** | §3.9 GitHub Actions → Cloudflare R2 异地冷备；每季度恢复演练 |
| CI/CD 改造后的首次 tag 把坏代码推到两个 prod | 中 | 高 | reusable ci-tests 必须含 `pnpm test`；首次改造后的第一次 tag 发一个 no-op release 验证链路 |

## 9. 停机时间预估

| 阶段 | 操作 | 估时 |
|---|---|---|
| T+0 | 停源 backend | 5s |
| T+1 | Final pg_dump + scp | 30s |
| T+2 | pg_restore 到 Railway | 60s |
| T+3 | 卷 tar / scp / 解压 | 60-120s（取决于大小） |
| T+4 | Railway backend redeploy | 60-90s（镜像 pull + migrate + start） |
| T+5 | 切 CNAME + DNS 收敛 | 60-120s |
| T+6 | 冒烟验证 | 180s |
| **合计** | | **7-12 分钟** |

## 10. 收尾

- [ ] 源端 Dokploy `gosealight-app-fkvf0t` 应用保留 **2 周** 冷备，不启动
- [ ] 2 周后确认无回滚需求 → Dokploy dashboard 删除
- [ ] GitHub variables 把 `DOKPLOY_ENABLED` 从 `true` 改成 `false`（彻底关掉 dokploy-deploy job）
- [ ] 更新 memory：
  - `reference_railway_and_dokploy.md` 加 `GoSeaLight-Self` 项目 ID 和 Deploy Hook 存放位置（只记"在 GitHub secrets 里"，不要写 URL）
  - `project_deploy_pipeline.md` 改："两个 Railway 项目 (LiuNian + Self) tag 触发统一发车，Deploy Hook 触发"
- [ ] 更新 `CLAUDE.md`：把 `home.365proxy.net` 改成 `light.gosea.in`，部署节换成 CI/CD 触发说明
- [ ] 更新 backend `.env.example`：确认 `APP_MASTER_KEY` 已在例子里（给新人看）
- [ ] 验证 branch protection（main 必须 PR + CI 绿才能 merge）已启用
- [ ] 验证 Railway 原生自动备份已启用（两个项目 dashboard 都要）
- [ ] 验证 R2 异地备份 workflow 跑了至少一次（手动 `workflow_dispatch` 触发验证）
- [ ] 写恢复演练 runbook（每季度执行一次，验证 R2 dump 真能 restore）
- [ ] 把改造前 LiuNian 冷备 dump 归档到 R2 单独目录 `archive/pre-cicd/`，永久保留

## 11. 命令附录（复制即用）

见 `docs/gosealight-self-migration-commands.sh`（本次不生成，等方案通过后再落）。
