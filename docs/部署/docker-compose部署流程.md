# docker-compose 部署

```bash

# 目录结构
prod
├── photography-server
├── photography-frontend
├── photography-wechat
├── photography-h5
├── volume # 存放挂载数据
├── .env
├── docker-compose.yml

# 0、先把四个仓的代码同步到本目录（compose 的挂载点直接指向仓库内文件，版本必须配套）
#   ./photography-server/scripts/nacos_init.sh                   -> nacos-init 容器执行体
#   ./photography-server/docs/sql/initdb/01-nacos-schema.sql    -> 首启建 nacos 库表（13 张）
#   ./photography-server/docs/sql/initdb/02-xxl-job-tables.sql  -> 首启建 xxl_job 库表
#   ./photography-server/docs/sql/ddl.sql / dml.sql             -> 首启建业务库表与初始数据
#   ./photography-server/config/nacos/*.yaml                    -> nacos-init 发布用的配置模板
# ⚠️ 缺这些文件时 Docker 会把【文件源】建成目录：mysql 初始化静默跳过、nacos-init 报 "is a directory"

# 1、把配置复制出来并修改成真实值
# 创建所有配置目录
mkdir -p volume/horizon volume/jaeger volume/nats/conf

# 复制，需要手动改值
cp ./photography-server/.env.example .env
cp ./photography-server/docker-compose.yml docker-compose.yml

# 复制根目录配置文件，需要手动改值
cp ./photography-server/config/horizon.example.yaml ./volume/horizon/horizon.yaml
cp ./photography-server/config/jaeger.example.yaml ./volume/jaeger/config.yaml
cp ./photography-server/config/nats.example.conf ./volume/nats/conf/nats.conf

# 2、在 prod 目录下执行
# 拉取所有 image 服务的最新镜像（镜像应在别处构建好再 push，2C2G 机器上 build 必 OOM）
docker compose pull

# 启动 —— 首次部署与日常启动是同一条命令，无需再手工分阶段初始化：
#   mysql 首启自动建好 photography / nacos / xxl_job 三个库与全部表（initdb，仅数据卷为空时执行）
#     -> nacos 就绪
#       -> 一次性容器 nacos-init 初始化管理员 + 发布业务配置（退出码 0）
#         -> backend -> frontend / h5 / wechat
docker compose up -d

# 想等所有健康检查通过再返回（适合脚本里判断"真的起完了"）
docker compose up -d --wait

# 【本机开发】改了代码需要重新构建镜像时
docker compose up -d --build backend frontend wechat h5

# 3、查看启动结果
docker compose ps -a                # nacos-init 显示 Exited (0) = 初始化成功
docker compose logs nacos-init      # 看它到底做了什么（登录 / 发布 / 跳过）

```

# 服务

| 服务                  | 容器名                             | 宿主机端口         | 说明                                           |
|---------------------|---------------------------------|---------------|----------------------------------------------|
| backend             | photography-backend             | 8080          | Go 后端 API                                    |
| frontend            | photography-frontend            | 8081          | PC 管理后台（Nginx 反代 `/api`）                     |
| h5                  | photography-h5                  | 8082          | H5 预约站（uni-app H5 产物）                        |
| wechat              | photography-wechat              | 8083          | 微信小程序员工端（uni-app 产物）                         |
| mysql               | photography-mysql               | 3306          | 数据库（photography 库）                           |
| redis               | photography-redis               | 6379          | 缓存                                           |
| nats                | photography-nats                | 4222 / 8222   | 消息队列 / 监控                                    |
| xxl-job-admin       | photography-xxl-job-admin       | 9100          | 任务调度中心                                       |
| elasticsearch       | photography-elasticsearch       | 9200          | 搜索引擎                                         |
| mongo               | photography-mongo               | 27017         | 文档数据库                                        |
| skywalking-oap      | photography-skywalking-oap      | 11800 / 12800 | 链路追踪后端①（skywalking-go native agent 上报 / 查询）  |
| skywalking-ui       | photography-skywalking-ui       | 9080          | 链路追踪 UI①（Horizon）                            |
| skywalking-banyandb | photography-skywalking-banyandb | 17912 / 17913 | 链路追踪存储①（BanyanDB）                            |
| jaeger              | photography-jaeger              | 4317 / 16686  | 链路追踪后端②（OTel OTLP 上报 / Jaeger UI）            |
| clickhouse          | photography-clickhouse          | 9000          | 链路追踪存储②（Jaeger 数据落库）                         |
| nacos               | photography-nacos               | 8848 / 9848   | 配置中心/注册中心（控制台 / SDK gRPC，9848=8848+1000 不可改）；存储已切 MySQL |
| nacos-init          | photography-nacos-init          | —             | **一次性容器**（跑完即退）：初始化管理员 + 发布 data_id，`Exited (0)` = 成功 |

后端容器的**业务配置**（db/redis/nats/mongodb/elasticsearch/xxljob/jwt/upload/log/jaeger）全部来自 Nacos，不再用 `APP_*`
环境变量注入，连接地址在 Nacos 模板里指向 compose 服务名；容器 env 只保留 `APP_PROFILE` + `APP_NACOS_*`（连接自举）+
`APP_CONFIG_SECRET*`（密文主密钥）。

---

# 依赖与初始化

启动不再需要人工分阶段，靠的是 `docker-compose.yml` 里两处声明：

1. **healthcheck + `condition: service_healthy`** —— 解决"容器起了但服务还不能用"。
   本仓 `mysql` / `redis` / `nats` / `nacos` 四个基础设施服务都配了真探针。
2. **一次性容器 + `condition: service_completed_successfully`** —— 解决"数据不存在"这类
   `depends_on` 天然表达不了的状态。

| 数据级依赖 | 由谁解决 |
|---|---|
| `photography` / `nacos` / `xxl_job` 三个库与全部表 | mysql 镜像的 `/docker-entrypoint-initdb.d/`（见 `docs/sql/initdb/`） |
| Nacos 管理员账号 | `nacos-init`（调 `POST /nacos/v3/auth/user/admin`，已存在则跳过） |
| Nacos 里的业务配置 `data_id` | `nacos-init`（与 `scripts/nacos_publish.sh` 同一套 v3 接口） |

完整的分层说明与排障见 [`README.md`](README.md)。

# Nacos 存储后端

`nacos` 服务已从「standalone + 内嵌 Derby」改为 **MySQL 存储**：

```
SPRING_DATASOURCE_PLATFORM=mysql        # 3.x 的键名（映射到 spring.sql.init.platform）
MYSQL_SERVICE_HOST/PORT/DB_NAME/USER/PASSWORD
MYSQL_SERVICE_DB_PARAM=...&allowPublicKeyRetrieval=true
```

- 原因：内嵌 Derby 落在**容器可写层**（该服务没有可写数据卷），`docker compose up -d --force-recreate`
  一次配置与注册信息就全丢。
- 库表由 `docs/sql/initdb/01-nacos-schema.sql` 建好（Nacos 3.2.4 官方 schema，13 张表）。
- ⚠️ 3.x 的配置项名是 `spring.sql.init.platform`，与 2.x 的 `spring.datasource.platform` 不同，勿照抄旧文档。
