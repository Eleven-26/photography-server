# SLOT 摄影工作室管理系统（后端）

面向摄影工作室的 SaaS 管理系统后端，提供 PC 管理后台、小程序管理后台、APP、H5 四类客户端共用的业务 API（RPC 风格）。

## 技术栈
- Go 1.26 + Gin
- MySQL 8（库：`photography`，utf8 / utf8_general_ci）
- GORM（soft_delete 软删除）、golang-jwt、viper
- Docker Compose 部署（MySQL / 后端 / 前端 分离）

## 目录结构
```
photography-server
├── cmd/server            # 入口 + 初始化引导（默认租户数据）
├── config                # 配置文件（config.yaml / config.example.yaml）
├── docs
│   ├── 需求文档-摄影工作室管理系统.md
│   └── sql               # DDL / DML 建库脚本
├── deploy                # 前端 nginx 部署配置
├── frontend              # 前端静态站点（基于首页高保真原型）
├── internal
│   ├── config            # 配置加载
│   ├── controller        # HTTP 控制器（控制层）
│   ├── database          # MySQL 连接
│   ├── middleware        # CORS / JWT / 日志 / 操作审计
│   ├── model             # 数据模型（统一 5 固定字段 + company_id）
│   ├── pkg               # logger / errs / jwtpkg
│   ├── response          # 统一响应
│   ├── router            # 路由（pc/miniapp/app/h5 分组）
│   └── service           # 业务服务层
├── uploads               # 上传文件目录（运行时生成）
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 快速开始（本地）
```bash
# 1. 初始化数据库（MySQL 8）
mysql -uroot -p < docs/sql/ddl.sql
mysql -uroot -p < docs/sql/dml.sql

# 2. 修改配置
cp config/config.example.yaml config/config.yaml   # 填数据库账号密码

# 3. 运行（默认 dev 环境）
go mod tidy
go run ./cmd/server -c config/config.yaml -p dev
```

不执行 DML 也可启动：服务首次启动会自动写入默认公司/门店/角色/超级管理员。

- 默认账号：`admin` / `admin123456`
- 健康检查：`GET /api/health`
- 登录：`POST /api/auth/login`

## 多环境配置
```
config/config.yaml          # 公共基础配置
config/config.dev.yaml      # 开发环境覆盖（默认）
config/config.test.yaml     # 测试环境覆盖
config/config.prod.yaml     # 生产环境覆盖
```
- 环境选择：启动参数 `-p dev|test|prod`，或环境变量 `APP_PROFILE`
- 配置优先级：`APP_*` 环境变量 > 环境配置文件 > `config.yaml`
- 配置文件中支持 `${VAR}` 占位符，自动用同名环境变量展开（如 `config.prod.yaml` 中的 `${APP_JWT_SECRET}`）
- 示例：
  - 开发：`go run ./cmd/server -c config/config.yaml -p dev`
  - 测试：`go run ./cmd/server -c config/config.yaml -p test`
  - 生产：`go run ./cmd/server -c config/config.yaml -p prod`

## Docker 部署
```bash
docker compose up -d --build
```
- MySQL：`localhost:3306`（photography 库，utf8）
- 后端 API：`http://localhost:8080/api/...`
- 前端站点：`http://localhost:8081/`（Nginx 反代 `/api` 到后端）

环境变量（可选，覆盖 config.yaml）：
`APP_DB_PASSWORD`、`APP_DB_HOST`、`APP_DB_PORT`、`APP_JWT_SECRET`、`APP_APP_PORT` 等（`APP_` 前缀 + 下划线组合）。

## 接口约定
- 统一响应：`{ "code": 0, "msg": "ok", "data": ... }`
- 认证：请求头 `Authorization: Bearer <token>`
- 路由：`POST /api/{pc|miniapp|app|h5}/{module}/{action}[/:id]`
- 完整接口清单见 `docs/需求文档-摄影工作室管理系统.md`

## 核心业务规则速览
- 定金 = 基础套餐价 × 定金比例；加选精修费全部计入尾款。
- 尾款 = 基础价 - 定金 + 加选；订单总额 = 基础价 + 加选。
- 退款按拍摄前小时数分档：≥72h 退 100%、48~72h 退 80%、24~48h 退 50%、<24h 不可退。
- 套餐被订单引用后改价自动生成新版本（历史订单快照一致）。
- 取消订单自动释放档期。