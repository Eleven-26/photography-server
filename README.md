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
- **链路追踪**：SkyWalking（go2sky gRPC 上报 OAP，gin 中间件自动埋点）
- **测试**：go-sqlmock（repository 单测，mock MySQL 连接，不依赖真实 DB）
- **其他**：golang-jwt（认证）、viper（多环境配置）
- **部署**：Docker Compose（MySQL / Redis / NATS / XXL-JOB / ES / MongoDB / SkyWalking / 后端 / 前端）

## 目录结构

```
photography-server
├── cmd
│   ├── server              # API 服务入口（配置加载 + 各组件初始化 + 优雅退出）
├── config
│   ├── config.yaml         # 公共基础配置
│   ├── config.dev.yaml     # 开发环境覆盖（默认）
│   ├── config.test.yaml    # 测试环境覆盖
│   ├── config.prod.yaml    # 生产环境覆盖
│   └── config.example.yaml # 示例配置模板
├── docs
│   ├── 需求文档-摄影工作室管理系统.md
│   └── sql                 # DDL / DML 建库脚本
├── internal
│   ├── common              # 通用常量（响应码 / 分页 / 上传）
│   ├── config              # 配置加载（多环境合并 + 环境变量展开）
│   ├── domain              # 领域纯函数（订单状态机 / 退款比例 / 编号生成 / 金额取整）
│   ├── enum                # 业务枚举（int 状态位）
│   ├── infrastructure      # 基础设施单例（MySQL/Redis/NATS/ES/MongoDB/XXL-JOB/SkyWalking）
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

## 多环境配置

加载机制：先加载基础 `config.yaml`，再用 `config.<profile>.yaml` 合并覆盖，最后环境变量兜底。

- 环境选择：启动参数 `-p dev|test|prod`，或环境变量 `APP_PROFILE`（默认 dev）
- 优先级：`APP_*` 环境变量 > 环境配置文件 > `config.yaml`
- 配置文件中支持 `${VAR}` 占位符，自动用同名环境变量展开（如 `config.prod.yaml` 中的 `${APP_JWT_SECRET}`）
- 示例：
  - 开发：`go run ./cmd/server -c config/config.yaml -p dev`
  - 测试：`go run ./cmd/server -c config/config.yaml -p test`
  - 生产：`go run ./cmd/server -c config/config.yaml -p prod`

主要配置段：`app` / `jwt` / `db`(MySQL) / `redis` / `nats` / `mongodb` / `log` / `upload` / `xxljob` / `elasticsearch`。

## Docker 部署
```
把photography-server的 .env.example 复制出来改成 .env（value要改成真实值）
把photography-server的 docker-compose.yml 复制出来

目录结构
pro
├── photography-server
├── photography-frontend
├── docker-compose.yml
├── .env

在pro目录下执行
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
| skywalking-oap | 11800 / 12800 | 链路追踪后端（agent 上报 / 查询） |
| skywalking-ui | 9080 | 链路追踪 UI |
| skywalking-banyandb | 17912 / 17913 | 链路追踪存储 |

后端容器内通过 `APP_*` 环境变量注入连接信息（见 `docker-compose.yml`），数据源均指向 compose 服务名。

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
| SkyWalking | `/test/skywalking/status` `/trace`（在请求链路下创建子 span 验证上报，数据到 UI 查看） |

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
