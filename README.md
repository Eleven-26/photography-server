# SLOT 摄影工作室管理系统（后端）

面向摄影工作室的 SaaS 管理系统后端，提供 PC 管理后台、小程序管理后台、APP、H5 四类客户端共用的业务 API（RPC 风格）。

## 技术栈

- **语言/框架**：Go 1.26 + Gin
- **数据库**：MySQL 8（GORM，soft_delete 软删除）
- **缓存**：Redis（go-redis v9）
- **消息队列**：NATS（含 JetStream 持久化）
- **搜索引擎**：Elasticsearch 8（go-elasticsearch v8）
- **文档数据库**：MongoDB（mongo-driver v2）
- **任务调度**：XXL-JOB
- **配置中心/注册中心**：Nacos（**唯一业务配置源，硬依赖**：本地仅留 bootstrap 连接段，业务配置 100% 托管 Nacos 按 data_id 分环境；拉取失败 fail-fast，SDK 本地快照兜底；实例自动注册/摘除）
- **链路追踪**（两通道各自独立，见「链路追踪」）：① SkyWalking Go agent（skywalking-go 编译期注入，直连 OAP native，Horizon「原生」模式 + 拓扑/指标分析）；② OpenTelemetry SDK → **Jaeger v2.18 + ClickHouse**（官方原生 ClickHouse 存储，Jaeger UI 按 trace_id 精确检索）。OTel 埋点代码为通道②专属（通道①由注入 agent 自动埋点，代码零侵入），切换仅改构建/部署配置
- **测试**：go-sqlmock（repository 单测，mock MySQL 连接，不依赖真实 DB）
- **其他**：golang-jwt（认证）、viper（多环境配置）
- **部署**：Docker Compose（MySQL / Redis / NATS / XXL-JOB / ES / MongoDB / SkyWalking / Nacos / 后端 / 前端）

## 目录结构

```
photography-server
├── cmd
│   ├── server              # API 服务入口（配置加载 + 各组件初始化 + 优雅退出）
├── config
│   ├── config.yaml         # Bootstrap（仅 Nacos 连接段，本地唯一配置文件）
│   ├── config.example.yaml # Bootstrap 模板
│   └── nacos               # Nacos 发布模板（控制台内容的版本化镜像，按 data_id 分环境）
│       ├── photography-server-dev.yaml
│       ├── photography-server-docker.dev.yaml
│       └── photography-server-prod.yaml
├── docs
│   ├── 需求文档-摄影工作室管理系统.md
│   └── sql                 # DDL / DML 建库脚本
├── internal
│   ├── common              # 通用常量（响应码 / 分页 / 上传）
│   ├── config              # 配置加载（多环境合并 + 环境变量展开）
│   ├── domain              # 领域纯函数（订单状态机 / 退款比例 / 编号生成 / 金额取整）
│   ├── enum                # 业务枚举（int 状态位）
│   ├── infrastructure      # 基础设施单例（MySQL/Redis/NATS/ES/MongoDB/XXL-JOB/Jaeger 通道/Nacos 注册）
│   ├── middleware          # CORS / JWT 认证 / 请求日志 / Recovery / 操作审计
│   ├── model               # 数据模型（统一 5 固定字段 + company_id 多租户）
│   ├── pkg                 # 基础能力包
│   │   ├── errs            # 错误类型 + 业务错误文案（统一出口）
│   │   ├── jwtpkg          # JWT 签发 / 校验
│   │   └── logger          # 日志封装
│   ├── presentation        # 外围接入层（HTTP / 定时任务 / 消息消费）
│   │   ├── controller      # HTTP 控制器
│   │   ├── dto             # 接口入参和出参的结构体
│   │   ├── job             # XXL-JOB 任务
│   │   └── mq              # NATS 消费者
│   ├── repository          # 数据访问层（WithTx 事务透传 + company_id 租户过滤）
│   ├── response            # 统一响应
│   ├── router              # 路由（pc/miniapp/app/h5 分组）
│   └── service             # 业务服务层（只经 repository/domain 访问数据，不直连基础设施）
├── uploads                 # 上传文件目录（运行时生成）
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .env.example            # 环境变量模板（复制为 .env 使用）
```

## 快速开始（本地）

```bash
# 1. 初始化数据库（MySQL 8）
mysql -uroot -p < docs/sql/ddl.sql
mysql -uroot -p < docs/sql/dml.sql

# 2. 运行（默认 dev 环境）
go mod tidy
go run ./cmd/server -c config/config.yaml -p dev
```

不执行 DML 也可启动：服务首次启动会自动写入默认公司/门店/角色/超级管理员（`cmd/server/bootstrap.go`）。

- 默认账号：`admin` / `admin123456`
- 健康检查：`GET /health`
- 登录：`POST /auth/login`

Makefile 常用命令：

```bash
make run      # 本地运行
make build    # 构建到 bin/
make test     # 跑测试
make tidy     # go mod tidy
make docker-up / docker-down / docker-build
```

## 配置加载（Nacos 单一配置源）

加载机制：本地 `config.yaml` 仅为 **bootstrap**（只含 `nacos.*` 连接段）→ 启动必拉 Nacos 远程配置（按 profile 区分 data_id）→ 环境变量最终覆盖。

- 环境选择：启动参数 `-p dev|test|prod`，或环境变量 `APP_PROFILE`（默认 dev）——只决定 data_id 中的 `${profile}` 占位
- 优先级：`APP_*` 环境变量 > Nacos 远程配置 > 本地 bootstrap
- 环境差异全部体现在 Nacos 上不同的 data_id（`photography-server-<profile>.yaml`，模板见 `config/nacos/`）
- 远程未发布的 key 即零值（仅 `app.port` / `jwt.expire_hours` / `upload.max_size_mb` 有代码兜底）；prod 下 `APP_JWT_SECRET` / `APP_DB_*` 仍强制 env 注入校验
- 变量命名规则：`APP_` + 段名_键名（`db.host`→`APP_DB_HOST`、`mongodb.*`→`APP_MONGODB_*`、`redis.addr`→`APP_REDIS_ADDR`、`app.mode`→`APP_APP_MODE`；回归测试见 `internal/config/env_mapping_test.go`）
- 示例：`go run ./cmd/server -c config/config.yaml -p dev`（需先本地起 Nacos 并发布 `photography-server-dev.yaml`）

主要配置段（都在 Nacos 上管理）：`app` / `jwt` / `db`(MySQL) / `redis` / `nats` / `mongodb` / `log` / `upload` / `xxljob` / `elasticsearch`。

## Docker 部署
```bash
# 把配置复制出来并修改成真实值

# 创建所有配置目录
mkdir -p volume/horizon volume/jaeger volume/nats/conf

# 复制，需要手动改值
cp ./photography-server/.env.example .env
cp ./photography-server/docker-compose.yml docker-compose.yml

# 复制根目录配置文件，需要手动改值
cp ./photography-server/config/horizon.example.yaml ./volume/horizon/horizon.yaml
cp ./photography-server/config/jaeger.example.yaml ./volume/jaeger/config.yaml
cp ./photography-server/config/nats.example.conf ./volume/nats/conf/nats.conf

# 目录结构
prod
├── photography-server
├── photography-frontend
├── volume # 存放挂载数据
├── docker-compose.yml
├── .env

# 在prod目录下执行
docker compose up -d --build
```

| 服务 | 宿主机端口 | 说明 |
|------|-----------|------|
| backend | 8080 | Go 后端 API |
| frontend | 8081 | 前端站点（Nginx 反代 `/api`） |
| mysql | 3306 | 数据库（photography 库） |
| redis | 6379 | 缓存 |
| nats | 4222 / 8222 | 消息队列 / 监控 |
| xxl-job-admin | 9100 | 任务调度中心 |
| elasticsearch | 9200 | 搜索引擎 |
| mongo | 27017 | 文档数据库 |
| skywalking-oap | 11800 / 12800 | 链路追踪后端①（skywalking-go native agent 上报 / 查询） |
| skywalking-ui | 9080 | 链路追踪 UI①（Horizon） |
| skywalking-banyandb | 17912 / 17913 | 链路追踪存储①（BanyanDB） |
| jaeger | 4317 / 16686 | 链路追踪后端②（OTel OTLP 上报 / Jaeger UI） |
| clickhouse | 9000 | 链路追踪存储②（Jaeger 数据落库） |
| nacos | 8848 / 9848 | 配置中心/注册中心（控制台 / SDK gRPC，9848=8848+1000 不可改） |

后端容器内通过 `APP_*` 环境变量注入连接信息（见 `docker-compose.yml`），数据源均指向 compose 服务名。

### 链路追踪（两通道各自独立）

两通道各自独立、互不依赖，**勿同时开启**（同一请求会产双 span / 双上报）：

| 通道 | 数据形态 | 启用方式 | 查看 |
|---|---|---|---|
| ① SkyWalking-go(native) | agent 编译期注入，直连 OAP:11800 | `.env` 设 `SW_AGENT_ENABLE=true` 构建（不注入 agent 则本通道不生效）；运行期无开关 | Horizon「原生」数据源 + 拓扑/指标 |
| ② OTel→Jaeger | OTLP → jaeger(collector+query 一体) → **ClickHouse**（v2.18.0 官方原生存储，alpha） | `.env` 设 `APP_JAEGER_ENABLE=true`、`APP_JAEGER_ENDPOINT=jaeger:4317`，**不注入 agent** | Jaeger UI :16686，**按 trace_id 精确检索** |

> `APP_JAEGER_*` 对应配置段 `jaeger.*`（OTel exporter 开关/地址）。通道②的 compose 服务：`docker compose up -d clickhouse jaeger`（先拷 `config/jaeger.example.yaml` → `./jaeger/config.yaml`）；ClickHouse 建库由 `CLICKHOUSE_DB=jaeger` 自动完成，Jaeger 侧 `create_schema: true` 自动建表。数据保留用 ClickHouse TTL（jaeger 配置 `ttl`）。

SkyWalking-go 版构建要点（Dockerfile 已内置开关 `SW_AGENT_ENABLE` / `SW_AGENT_VERSION` / `SW_AGENT_SERVICE` / `SW_AGENT_BACKEND`；agent 二进制本地化在 `build/agent/`，**有意随仓库入库约 45MB**——Dockerfile 为离线构建，新环境 clone 后直接 COPY 进镜像即可，无需联网下载）：

```bash
# 本机注入构建（需先下载对应版本 agent 二进制）
go build -toolexec="<agent-路径> -config <agent.config>" -a -o server ./cmd/server
# agent.config 示例：reporter.grpc.backend_service 指向 OAP:11800（Dockerfile 内自动生成）
docker compose build backend   # SW_AGENT_ENABLE=true 时产物自动织入 agent
```

代码侧：SkyWalking-go 由 agent 自动埋点 gin HTTP 入口与 gorm SQL，无需业务埋点；通道②（Jaeger）复用 OTel 手动埋点（gin otelgin / gorm OTel 插件 / xxl-job 根 span / NATS traceparent 透传），由 `APP_JAEGER_ENABLE` 控制。响应 `trace_id` 双通道通用：Jaeger 版取 OTel entry span，native 版取 agent native trace id（自动切换取值源，无感知）。xxl-job / NATS 的手动埋点暂仅 OTel（Jaeger）通道生效（SkyWalking-go native 版为 P1 待办，当前 HTTP+SQL 主链路已覆盖）。

### Nacos 配置中心 + 服务注册（唯一配置源，硬依赖）

无开关：本地只保留 bootstrap（`config.yaml`，仅 `nacos.*` 连接段），业务配置 100% 托管 Nacos。

- 优先级 `APP_* 环境变量 > Nacos 远端 > 本地 bootstrap`；拉取失败直接终止启动（fail-fast）；Nacos 短暂不可用时 SDK 自动读本地快照兜底（快照目录已挂载持久化）
- 环境差异用不同 data_id：`photography-server-dev.yaml` / `photography-server-docker.dev.yaml` / `photography-server-prod.yaml`（Group `DEFAULT_GROUP`）

行为细节：

| 能力 | 说明 |
|---|---|
| 配置拉取 | 启动时 `GetConfig` 全量拉取合并；**配置变更需重启进程生效**（不做运行期热更） |
| 服务注册 | HTTP 端口就绪后注册为**临时实例**（SDK 自动心跳，进程退出自动摘除；优雅退出时另有主动反注册），metadata 带 `profile` |
| 快照目录 | `.nacos/cache`（`nacos.cache_dir` 可改，compose 已挂载持久卷）；SDK 日志在 `.nacos/log` |

启用步骤：`docker compose up -d nacos` → 控制台 `http://localhost:8850/`（v3 独立端口，默认 admin/nacos，开启鉴权后用 `.env` 里配置的账号）→ 按 `config/nacos/photography-server-<profile>.yaml` 模板新建 YAML 配置并发布 → 确认 `.env` 的 `APP_NACOS_SERVER_ADDR/USERNAME/PASSWORD` 后启动 backend（或整栈 `docker compose up -d`，nacos 为硬依赖会先起）。

**配置推送脚本**（替代控制台手贴，模板即唯一来源）：

```bash
./scripts/nacos_publish.sh dev --dry-run   # 试运行：只校验模板与鉴权，不推送
./scripts/nacos_publish.sh dev             # 推送 config/nacos/photography-server-dev.yaml
./scripts/nacos_publish.sh prod            # 其他环境：docker.dev / test / prod
```

- 脚本自动加载项目根 `.env`：`NACOS_PUBLISH_SERVER`、`APP_NACOS_USERNAME/PASSWORD`、`APP_NACOS_IDENTITY_KEY/VALUE`（token 鉴权被拒时自动降级）、`APP_NACOS_NAMESPACE/GROUP`
- 推送 = 用模板内容**整体覆盖**远端该 data_id（`type=yaml`，归属应用 `appName` 取 `APP_NACOS_APP_NAME`，默认 `photography-server`）；发布前确认模板已 git commit
- 控制台新建配置时「归属应用」填同一个 `photography-server`，保持与脚本推送的元数据一致

## 接口约定

- 统一响应：`{ "code": 0, "msg": "ok", "data": ... }`
- 认证：请求头 `Authorization: Bearer <token>`
- 路由风格：`POST /{pc|miniapp|app|h5}/{module}/{action}[/:id]`（业务接口均需 JWT）
- 完整接口清单见 `docs/需求文档-摄影工作室管理系统.md`

### 调试接口（`/test/*`，仅 dev/test 注册）

用于验证各基础设施连通性：直连基础设施单例、不挂业务鉴权；路由仅在非 `release` 模式注册，生产自动下线。

| 模块 | 接口 |
|------|------|
| Redis | `/test/redis/ping` `/set` `/get` `/del` |
| NATS | `/test/nats/status` `/pub` `/pub-persistent` `/pub-pull` `/request` |
| Elasticsearch | `/test/es/status` `/index` `/search` `/list` `/delete` |
| MongoDB | `/test/mongo/status` `/insert` `/insert-many` `/find` `/find-one` `/update` `/delete` `/delete-by-id` |
| Jaeger 通道 | `/test/jaeger/status` `/trace`（在请求链路下创建子 span 验证上报，数据到 Jaeger UI 查看） |

## 单元测试

测试与被测代码同目录同包放置（Go 惯例，白盒可测未导出实现），`make test` 即 `go test ./...`：

| 包 | 文件 | 覆盖 |
|---|---|---|
| `internal/domain` | `domain_test.go` | 订单状态机流转 / 回退边界、退款四档与临界时间、金额取整、编号格式 |
| `internal/repository` | `order_repo_test.go`、`base_test.go` | 事务 WithTx 提交与回滚、CAS 条件更新、company_id 租户过滤、行锁读（FOR UPDATE）、分页归一化 |

- repository 测试经 go-sqlmock 注入 mock MySQL 连接（`newMockRepo` 白盒构造），不依赖真实数据库；
- 大量静态 mock 数据（如 JSON fixture）按 Go 惯例放各包下 `testdata/`（工具链自动忽略、测试以相对路径读取），无需另建 test 目录；
- 需真实中间件、不进主链路的集成 / E2E 测试，才建议独立目录 + build tag（如 `test/integration`），当前仓库无此场景。

## 核心业务规则速览

- 定金 = 基础套餐价 × 定金比例；加选精修费全部计入尾款。
- 尾款 = 基础价 - 定金 + 加选；订单总额 = 基础价 + 加选。
- 退款按拍摄前小时数分档：≥72h 退 100%、48~72h 退 80%、24~48h 退 50%、<24h 不可退。
- 套餐被订单引用后改价自动生成新版本（历史订单快照一致）。
- 取消订单自动释放档期。
- 订单状态机 / 退款分档 / 编号生成 / 金额取整等纯业务规则集中在 `internal/domain`（零依赖、可独立单测），service 只做编排。
- 多租户：数据访问全部收敛在 repository 层并按 `company_id` 过滤；事务以 `repository.Tx` 为唯一入口，service 不持有数据库句柄。
