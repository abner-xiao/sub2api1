# Sub2API Render 部署指南

本文档面向仓库 `https://github.com/abner-xiao/sub2api1`，说明如何把 Sub2API 部署到 Render，并明确 Render 控制台配置项、项目环境变量、可选配置文件修改点，以及部署后的验证方式。

> 验证日期：2026-06-13  
> 验证仓库提交：`e34ad2b1`  
> 本地已验证：前端生产构建通过；后端 `go build -tags embed` 通过；配置和自动初始化单元测试通过。  
> 未在本机验证：Docker 镜像构建和 Render 真实上线部署，因为当前环境没有 Docker，也没有 Render 账号权限。

## 1. 部署架构

推荐使用 Render 的 3 个托管资源：

| 资源 | Render 服务类型 | 用途 |
| --- | --- | --- |
| `sub2api` | Web Service，Runtime 选择 Docker | 运行 Sub2API 后端和内嵌前端 |
| `sub2api-db` | Render Postgres | 业务数据库 |
| `sub2api-redis` | Render Key Value | Redis/Valkey 兼容缓存和队列 |

项目根目录已有 `Dockerfile`，它会：

1. 使用 Node/pnpm 构建 `frontend`。
2. 把前端产物写入 `backend/internal/web/dist`。
3. 使用 Go `-tags embed` 构建后端。
4. 运行 `/app/sub2api`，默认监听 `SERVER_PORT`。

Render 官方要求 Web Service 监听 `0.0.0.0` 上的 HTTP 端口，默认端口是 `10000`。本项目读取的是 `SERVER_HOST` 和 `SERVER_PORT`，不是直接读取 `PORT`，因此 Render 上必须显式设置：

```text
SERVER_HOST=0.0.0.0
SERVER_PORT=10000
```

## 2. 项目中需要关注的配置位置

### 2.1 根目录 Dockerfile

文件：`Dockerfile`

Render 应使用仓库根目录的 `Dockerfile`：

```text
Dockerfile Path: ./Dockerfile
Docker Build Context Directory: .
```

通常不需要修改 Dockerfile。它已经：

- 暴露 `8080`，但实际运行端口由 `SERVER_PORT` 环境变量控制。
- 健康检查访问 `http://localhost:${SERVER_PORT:-8080}/health`。
- 把 `/app/data` 作为数据目录。

### 2.2 环境变量优先于配置文件

文件：`backend/internal/config/config.go`

项目使用 Viper 读取配置，并把点号配置映射为下划线环境变量，例如：

```text
server.port        -> SERVER_PORT
database.host      -> DATABASE_HOST
redis.enable_tls   -> REDIS_ENABLE_TLS
jwt.secret         -> JWT_SECRET
```

Render 部署时推荐通过环境变量配置，不需要把 `config.yaml` 提交到仓库。

### 2.3 自动初始化

文件：`backend/internal/setup/setup.go`

Docker 部署推荐开启：

```text
AUTO_SETUP=true
```

首次启动时项目会从环境变量读取 PostgreSQL、Redis、管理员账号、JWT 等配置，初始化数据库并写入 `/app/data/config.yaml` 和安装锁文件。Render 上建议给 `/app/data` 挂载 Persistent Disk，否则实例重建后本地生成的 `config.yaml`、锁文件、文件日志等会丢失。

重要：数据库数据在 Render Postgres 中，不依赖 `/app/data`；但 `/app/data` 仍建议持久化，避免每次新实例都重复执行安装检测。

## 3. Render 手动部署步骤

### 3.1 创建 Render Postgres

在 Render Dashboard：

1. New -> Postgres。
2. Name：`sub2api-db`。
3. Region：选择与 Web Service 相同的区域。
4. PostgreSQL Version：建议 `18`；如果你的 Render 控制台暂不支持，可选当前可用最新稳定版本。
5. Database：`sub2api`。
6. User：`sub2api`。
7. Plan：生产建议至少 Starter/付费计划；测试可用免费/低配计划。
8. IP Allow List：如果只允许 Render 内部服务访问，设置为空列表或仅内部访问；不要开放到公网，除非你确实需要本地直连调试。

创建后在连接信息中记录：

| Sub2API 环境变量 | Render Postgres 字段 |
| --- | --- |
| `DATABASE_HOST` | Internal Hostname |
| `DATABASE_PORT` | Port，通常 `5432` |
| `DATABASE_USER` | User |
| `DATABASE_PASSWORD` | Password |
| `DATABASE_DBNAME` | Database |
| `DATABASE_SSLMODE` | 建议 `require` 或 `prefer` |

### 3.2 创建 Render Key Value

在 Render Dashboard：

1. New -> Key Value。
2. Name：`sub2api-redis`。
3. Region：必须与 Web Service 相同。
4. Plan：生产建议付费计划，因为免费 Key Value 不保证持久化。
5. Maxmemory Policy：一般选择 `allkeys-lru` 或按业务需要选择。
6. External Connections：默认不启用即可。

Sub2API 不能直接读取 `REDIS_URL`，需要把 Render Key Value 的 Internal URL 拆成环境变量：

内部 URL 常见格式：

```text
redis://red-xxxx:6379
```

对应：

```text
REDIS_HOST=red-xxxx
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_ENABLE_TLS=false
```

如果你开启了 Render Key Value 的 Internal Authentication，内部 URL 会包含用户名和密码，例如：

```text
redis://default:PASSWORD@red-xxxx:6379
```

此时配置：

```text
REDIS_HOST=red-xxxx
REDIS_PORT=6379
REDIS_PASSWORD=PASSWORD
REDIS_DB=0
REDIS_ENABLE_TLS=false
```

Render Key Value 的外部连接通常使用 TLS，但 Web Service 同区域内部连接使用 Internal URL，默认不需要 TLS。

### 3.3 创建 Web Service

在 Render Dashboard：

1. New -> Web Service。
2. Connect Repository：选择 `abner-xiao/sub2api1`。
3. Branch：选择要部署的分支，例如 `main`。
4. Runtime / Language：选择 `Docker`。
5. Name：`sub2api`。
6. Region：必须与 Postgres、Key Value 相同。
7. Root Directory：留空。
8. Dockerfile Path：`./Dockerfile`。
9. Docker Context Directory：`.`。
10. Instance Type：生产建议至少 1 CPU / 1 GB 内存；构建期也需要足够内存，低配可能因前端/Go 构建失败。
11. Health Check Path：`/health`。
12. Auto-Deploy：按需开启。生产环境建议先关闭，确认配置后手动部署；稳定后再开启。

### 3.4 Web Service 环境变量

Render Web Service -> Environment 添加下列变量。

#### 必填

| Key | Value | 说明 |
| --- | --- | --- |
| `AUTO_SETUP` | `true` | 首次启动自动初始化 |
| `SERVER_HOST` | `0.0.0.0` | Render 要求监听所有网卡 |
| `SERVER_PORT` | `10000` | Render Web Service 默认端口 |
| `SERVER_MODE` | `release` | 生产模式 |
| `RUN_MODE` | `standard` | 完整 SaaS 功能 |
| `DATABASE_HOST` | Postgres Internal Hostname | 不要填 External Hostname |
| `DATABASE_PORT` | `5432` | Postgres 端口 |
| `DATABASE_USER` | `sub2api` 或 Render 生成用户 | 数据库用户 |
| `DATABASE_PASSWORD` | Render Postgres 密码 | 标记为 Secret |
| `DATABASE_DBNAME` | `sub2api` 或 Render 生成数据库名 | 数据库名 |
| `DATABASE_SSLMODE` | `require` 或 `prefer` | Render 托管 Postgres 建议开启 SSL |
| `REDIS_HOST` | Key Value Internal Host | 从 Internal URL 拆出 |
| `REDIS_PORT` | `6379` | Redis/Valkey 端口 |
| `REDIS_PASSWORD` | 空或 Internal Auth 密码 | 开启内部认证才填 |
| `REDIS_DB` | `0` | 默认 DB |
| `REDIS_ENABLE_TLS` | `false` | 同区内部连接通常不使用 TLS |
| `ADMIN_EMAIL` | 你的管理员邮箱 | 首次启动创建管理员 |
| `ADMIN_PASSWORD` | 强密码 | 标记为 Secret；留空会自动生成但不利于 Render 日志后续查找 |
| `JWT_SECRET` | 64 位十六进制随机字符串 | 必须固定，避免重启后登录失效 |
| `TOTP_ENCRYPTION_KEY` | 64 位十六进制随机字符串 | 必须固定，避免 2FA/加密数据失效 |
| `TZ` | `Asia/Shanghai` 或你的时区 | 影响日志和统计日期 |

生成密钥示例：

```bash
openssl rand -hex 32
```

#### 推荐

| Key | Value | 说明 |
| --- | --- | --- |
| `LOG_FORMAT` | `json` | Render 日志更易检索 |
| `LOG_OUTPUT_TO_STDOUT` | `true` | 输出到 Render Logs |
| `LOG_OUTPUT_TO_FILE` | `false` | Render 上通常不需要写文件日志 |
| `DATABASE_MAX_OPEN_CONNS` | `20` 到 `50` | 按 Render Postgres 规格调整，避免连接过多 |
| `DATABASE_MAX_IDLE_CONNS` | `5` 到 `10` | 小规格数据库建议降低 |
| `DATABASE_CONN_MAX_LIFETIME_MINUTES` | `30` | 保持默认即可 |
| `DATABASE_CONN_MAX_IDLE_TIME_MINUTES` | `5` | 保持默认即可 |
| `REDIS_POOL_SIZE` | `50` 到 `200` | 按 Key Value 规格调整 |
| `REDIS_MIN_IDLE_CONNS` | `5` 到 `20` | 小规格建议降低 |

#### 按需

| Key | 何时设置 |
| --- | --- |
| `SERVER_FRONTEND_URL` | 有自定义域名或需要邮件重置链接时设置，例如 `https://sub2api.example.com` |
| `CORS_ALLOWED_ORIGINS` | 前后端分离或第三方站点跨域访问时设置；当前 Docker 内嵌前端通常不需要 |
| `UPDATE_PROXY_URL` | Render 出站访问 GitHub 慢或需要代理时设置 |
| `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET` | 需要 Gemini OAuth 时设置 |
| `GEMINI_CLI_OAUTH_CLIENT_SECRET` | 需要内置 Gemini CLI OAuth 时设置 |
| `ANTIGRAVITY_OAUTH_CLIENT_SECRET` | 需要 Antigravity OAuth 时设置 |

## 4. Persistent Disk 配置

Web Service -> Disks 添加：

```text
Name: sub2api-data
Mount Path: /app/data
Size: 1 GB 起
```

说明：

- `/app/data/config.yaml` 和安装锁文件会写在这里。
- 如果 `LOG_OUTPUT_TO_FILE=true`，日志也会写在这里。
- Render 带 Persistent Disk 的服务通常不能水平扩容到多实例；Sub2API 使用共享数据库和 Redis，但本地 `/app/data` 仍是单实例持久盘模型。生产扩容前需要重新评估配置文件、日志和本地文件依赖。

## 5. `render.yaml` Blueprint 示例

如果你希望用 Infrastructure as Code 管理 Render，可以在仓库根目录新增 `render.yaml`。下面是可用模板，需要根据 Render 当前套餐和区域调整 `plan`、`region`。

注意：

- Render Blueprint 支持从 Postgres 使用 `fromDatabase` 引用 `host`、`port`、`user`、`password`、`database` 等属性。
- Render Blueprint 对 Key Value 可靠暴露的是 `connectionString`，格式类似 `redis://red-xxxx:6379`。当前 Sub2API 不能直接读取 `REDIS_URL`，也不能在 Blueprint 中做字符串拆分，所以 `REDIS_HOST` 需要在首次 Blueprint 同步后从 Key Value Internal URL 手动填写，或先修改项目代码支持解析 `REDIS_URL`。
- 示例中 `ADMIN_PASSWORD`、`JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`REDIS_HOST` 使用 `sync: false`，表示需要在 Render Dashboard 里手动填入 Secret/值。

```yaml
services:
  - type: web
    name: sub2api
    runtime: docker
    repo: https://github.com/abner-xiao/sub2api1
    branch: main
    region: singapore
    plan: starter
    dockerfilePath: ./Dockerfile
    dockerContext: .
    healthCheckPath: /health
    autoDeploy: false
    disk:
      name: sub2api-data
      mountPath: /app/data
      sizeGB: 1
    envVars:
      - key: AUTO_SETUP
        value: "true"
      - key: SERVER_HOST
        value: 0.0.0.0
      - key: SERVER_PORT
        value: "10000"
      - key: SERVER_MODE
        value: release
      - key: RUN_MODE
        value: standard
      - key: TZ
        value: Asia/Shanghai
      - key: DATABASE_HOST
        fromDatabase:
          name: sub2api-db
          property: host
      - key: DATABASE_PORT
        fromDatabase:
          name: sub2api-db
          property: port
      - key: DATABASE_USER
        fromDatabase:
          name: sub2api-db
          property: user
      - key: DATABASE_PASSWORD
        fromDatabase:
          name: sub2api-db
          property: password
      - key: DATABASE_DBNAME
        fromDatabase:
          name: sub2api-db
          property: database
      - key: DATABASE_SSLMODE
        value: require
      - key: REDIS_HOST
        sync: false
      - key: REDIS_PORT
        value: "6379"
      - key: REDIS_PASSWORD
        value: ""
      - key: REDIS_DB
        value: "0"
      - key: REDIS_ENABLE_TLS
        value: "false"
      - key: ADMIN_EMAIL
        value: admin@example.com
      - key: ADMIN_PASSWORD
        sync: false
      - key: JWT_SECRET
        sync: false
      - key: TOTP_ENCRYPTION_KEY
        sync: false
      - key: LOG_FORMAT
        value: json
      - key: LOG_OUTPUT_TO_STDOUT
        value: "true"
      - key: LOG_OUTPUT_TO_FILE
        value: "false"
      - key: DATABASE_MAX_OPEN_CONNS
        value: "50"
      - key: DATABASE_MAX_IDLE_CONNS
        value: "10"
      - key: REDIS_POOL_SIZE
        value: "200"
      - key: REDIS_MIN_IDLE_CONNS
        value: "10"

  - type: keyvalue
    name: sub2api-redis
    region: singapore
    plan: starter
    ipAllowList: []

databases:
  - name: sub2api-db
    region: singapore
    plan: basic-256mb
    postgresMajorVersion: "18"
    databaseName: sub2api
    user: sub2api
    ipAllowList: []
```

如果你的工作区仍使用旧套餐名，或者 Render 控制台展示的套餐名不同，请按 Dashboard 可选项调整 `plan`。Postgres 新实例不再建议使用 legacy `starter`，示例使用当前 Blueprint 文档列出的 `basic-256mb`。

## 6. 必须标明的项目配置修改点

### 6.1 不建议提交 `config.yaml`

不要把生产密钥写入仓库。Render 上优先用环境变量。`deploy/config.example.yaml` 只作为参考。

### 6.2 如需修改配置文件

如果你坚持使用配置文件，可复制：

```bash
cp deploy/config.example.yaml config.yaml
```

然后至少修改：

```yaml
server:
  host: "0.0.0.0"
  port: 10000
  mode: "release"
  frontend_url: "https://你的-render域名或自定义域名"

database:
  host: "Render Postgres Internal Hostname"
  port: 5432
  user: "sub2api"
  password: "Render Postgres Password"
  dbname: "sub2api"
  sslmode: "require"

redis:
  host: "Render Key Value Internal Host"
  port: 6379
  password: ""
  db: 0
  enable_tls: false

jwt:
  secret: "openssl rand -hex 32 生成的固定值"

totp:
  encryption_key: "openssl rand -hex 32 生成的固定值"
```

但 Docker 镜像默认会从 `/app/data/config.yaml`、当前目录和 `/etc/sub2api` 查找配置文件。Render 上若要挂载配置文件，需要将文件放到 `/app/data/config.yaml`。Render 的 Web Service 不适合直接在控制台上传复杂配置文件，因此更推荐环境变量。

### 6.3 端口必须改为 Render 端口

不要保持 Docker Compose 示例里的：

```text
SERVER_PORT=8080
```

Render 上必须设置为：

```text
SERVER_PORT=10000
```

否则 Render 可能无法正确探测到服务，健康检查会失败。

### 6.4 数据库和 Redis 不能只填 URL

`deploy/DOCKER.md` 中有 `DATABASE_URL` 和 `REDIS_URL` 示例，但当前代码实际读取的是拆分后的字段：

```text
DATABASE_HOST
DATABASE_PORT
DATABASE_USER
DATABASE_PASSWORD
DATABASE_DBNAME
DATABASE_SSLMODE

REDIS_HOST
REDIS_PORT
REDIS_PASSWORD
REDIS_DB
REDIS_ENABLE_TLS
```

在 Render 控制台不要只设置 `DATABASE_URL` / `REDIS_URL`，否则应用会继续尝试连接默认的 `localhost`。

## 7. 部署验证

### 7.1 本地已完成的编译验证

本地环境：

```text
Node.js v20.20.2
pnpm v9.15.9 via Corepack
Go 1.26.4 临时工具链
```

执行并通过：

```bash
cd frontend
corepack pnpm install --frozen-lockfile
corepack pnpm run build
```

结果：前端生产构建成功，产物写入 `backend/internal/web/dist`。

执行并通过：

```bash
cd backend
GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn go mod download
CGO_ENABLED=0 GOOS=linux go build -tags embed -trimpath -o ../.tmp/sub2api ./cmd/server
```

结果：后端 Linux 二进制构建成功。

执行并通过：

```bash
cd backend
go test ./internal/config
go test ./internal/setup
```

结果：环境变量配置映射和自动初始化相关单元测试通过。

### 7.2 本地 Docker 验证命令

如果你的本机有 Docker，可执行：

```bash
docker build -t sub2api-render-test .
```

再用本地 PostgreSQL/Redis 或 Docker Compose 验证启动：

```bash
docker run --rm \
  -p 10000:10000 \
  -e AUTO_SETUP=true \
  -e SERVER_HOST=0.0.0.0 \
  -e SERVER_PORT=10000 \
  -e SERVER_MODE=release \
  -e DATABASE_HOST=你的数据库地址 \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=sub2api \
  -e DATABASE_PASSWORD=你的数据库密码 \
  -e DATABASE_DBNAME=sub2api \
  -e DATABASE_SSLMODE=require \
  -e REDIS_HOST=你的Redis地址 \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD= \
  -e REDIS_DB=0 \
  -e REDIS_ENABLE_TLS=false \
  -e ADMIN_EMAIL=admin@example.com \
  -e ADMIN_PASSWORD=强密码 \
  -e JWT_SECRET=openssl_rand_hex_32生成值 \
  -e TOTP_ENCRYPTION_KEY=openssl_rand_hex_32生成值 \
  sub2api-render-test
```

健康检查：

```bash
curl -i http://localhost:10000/health
```

预期 HTTP 状态码：`200`。

### 7.3 Render 上线后验证

假设 Render 分配域名：

```text
https://sub2api.onrender.com
```

执行：

```bash
curl -i https://sub2api.onrender.com/health
```

预期：

```text
HTTP/2 200
```

然后访问：

```text
https://sub2api.onrender.com
```

用 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 登录后台。

在 Render Logs 中确认出现以下关键日志：

```text
Auto setup mode enabled...
Database connection successful
Redis connection successful
Database initialized successfully
Auto setup completed successfully!
```

如果已有 `/app/data` 安装锁文件，后续重启不一定重复出现 auto setup 日志，这是正常现象。

## 8. 常见问题

### 8.1 Deploy 后一直显示 unhealthy

检查：

1. `SERVER_HOST` 是否为 `0.0.0.0`。
2. `SERVER_PORT` 是否为 `10000`。
3. Render Health Check Path 是否为 `/health`。
4. Logs 是否有数据库或 Redis 连接失败。

### 8.2 数据库连接失败

检查：

1. Web Service、Postgres 是否同 Region。
2. 是否使用 Internal Hostname。
3. `DATABASE_SSLMODE` 是否设置为 `require` 或 `prefer`。
4. 用户名、密码、数据库名是否对应 Render Postgres 的实际值。

### 8.3 Redis 连接失败

检查：

1. Web Service、Key Value 是否同 Region。
2. `REDIS_HOST` 是否从 Internal URL 拆出，不要带 `redis://`。
3. `REDIS_PORT` 是否为 `6379`。
4. 如果开启 Internal Authentication，`REDIS_PASSWORD` 是否填写。
5. 内部连接通常 `REDIS_ENABLE_TLS=false`。

### 8.4 重启后用户被迫重新登录

`JWT_SECRET` 未固定。设置固定的 64 位十六进制随机值后重新部署。

### 8.5 开启 2FA 后重启无法验证

`TOTP_ENCRYPTION_KEY` 未固定。必须设置固定值；如果之前已经用随机值加密了 2FA 数据，可能需要重新配置用户 2FA。

### 8.6 构建时内存不足

Dockerfile 会同时涉及 Node 前端构建和 Go 后端构建。Render 低配实例可能构建失败。解决方案：

1. 升级 Web Service 实例类型。
2. 使用预构建镜像推送到镜像仓库，再在 Render 选择 Existing Image。
3. 减少前端构建并发或优化构建产物，需改项目配置。

## 9. 官方参考

- Render Web Services: https://render.com/docs/web-services
- Render Docker: https://render.com/docs/docker
- Render Blueprint YAML Reference: https://render.com/docs/blueprint-spec
- Render Key Value: https://render.com/docs/key-value
- Render Postgres: https://render.com/docs/postgresql
