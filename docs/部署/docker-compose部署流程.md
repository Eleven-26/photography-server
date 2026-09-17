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

# 2、在prod目录下执行
# 拉取所有 image 服务的最新镜像
docker compose pull

# 重新构建所有 build 服务，并启动
docker compose up -d --build

# 或
# 指定构建镜像，发代码需指定更新
docker compose up -d --build backend frontend

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
| nacos               | photography-nacos               | 8848 / 9848   | 配置中心/注册中心（控制台 / SDK gRPC，9848=8848+1000 不可改） |

后端容器的**业务配置**（db/redis/nats/mongodb/elasticsearch/xxljob/jwt/upload/log/jaeger）全部来自 Nacos，不再用 `APP_*`
环境变量注入，连接地址在 Nacos 模板里指向 compose 服务名；容器 env 只保留 `APP_PROFILE` + `APP_NACOS_*`（连接自举）+
`APP_CONFIG_SECRET*`（密文主密钥）。
