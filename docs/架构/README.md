# 架构文档（photography 四仓）

> 本目录是 SLOT 摄影工作室管理系统的**跨仓架构基线**，统一维护在 `photography-server`。
> 四个仓库独立部署、共同演进，但共享一套接口契约与设计口径；改动架构时请同步更新本目录。

## 一、四个仓库

| 仓库 | 角色 | 技术栈 | 对外端口（compose） |
| --- | --- | --- | --- |
| `photography-server` | 后端 API（唯一数据源） | Go 1.26 · Gin · GORM · MySQL/Redis/NATS/ES/Mongo | 8080 |
| `photography-frontend` | PC 管理后台（员工） | Vue 3.5 + TypeScript + Vite 6 + Pinia 3 | 8081 |
| `photography-wechat` | 员工端小程序（兼 H5） | uni-app Vue3（纯 JS） | 8083 |
| `photography-h5` | 客户端 H5（兼小程序） | uni-app Vue3（纯 JS） | 8082 |

## 二、文档索引

| 文档 | 内容 |
| --- | --- |
| [01-目录地图](./01-目录地图.md) | 四仓目录树、逐目录职责、文件放置约定 |
| [02-系统架构图](./02-系统架构图.md) | 总体架构、请求链路、分层、部署拓扑（Mermaid） |
| [03-结构审视与整改](./03-结构审视与整改.md) | 四仓结构问题分级、本次已整改项、待办建议 |
## 三、维护约定

- 本目录是**四仓结构的事实来源**；各仓 `README.md` 只保留“如何跑”，架构细节指回本目录相对路径 `../photography-server/docs/架构/`。
- 新增/移动目录、调整分层、变更部署拓扑时，在同一个变更内同步更新 `01` 与 `02`。
- 结构问题清单（`03`）按 P0/P1/P2 维护；已修复项标注日期与范围，未修复项给出建议改法。
- 后端接口清单以声明式路由表为准：`internal/presentation/routes/` + `internal/router/endpoints.go`；Swagger 见 `docs/swagger/`。