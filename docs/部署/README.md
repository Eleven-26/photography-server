# 服务依赖

![img.png](图片/img.png)
其实混了三种性质不同的依赖，导致无法一键启动：

| 类型         | 例子                                      | compose 能否表达                    |
|------------|-----------------------------------------|---------------------------------|
| ① 进程级（顺序）  | redis 先于 backend                        | ✅ depends_on 裸写                 |
| ② 就绪级（能服务） | 	mysql 能 ping 通                         | ✅ healthcheck + service_healthy |
| ③ 数据级	     | Nacos 里有 data_id、MySQL 里有表、Nacos 里有管理员	 | ❌ compose 无权表达                  

所以答案是：把第③类也变成编排里的一环——用一次性 init 容器（depends_on: condition: service_completed_successfully）。补齐后
docker compose up -d 就真的是全自动。

# 具体方案

需要分阶段部署

## 1、先部署需要初始化数据的服务

| 服务    | 初始化                                               | 依赖      
|-------|---------------------------------------------------|---------|
| mysql | 1、初始化nacos数据表<br/>2、初始化xxl-job数据表<br/>3、初始化业务表和数据 |         |
| nacos | 初始化业务配置                                           | 依赖mysql |

## 2、部署其他中间件服务

redis、nats、clickhouse、jaeger、mongo、elasticsearch

## 3、部署应用服务

backend、frontend、wechat、h5