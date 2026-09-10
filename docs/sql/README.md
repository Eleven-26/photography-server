# 数据库脚本规范（docs/sql）

多租户 SaaS，**单库多租户**（`company_id` 隔离）。

- **新环境 / 新租户** → 用 **全量脚本** 一次性初始化
- **已上线环境** → 用 **增量脚本** 按日期顺序升级

**核心不变量：全量脚本单独执行一次的结果 == 增量目录下所有脚本按顺序依次执行的结果。**
每次改动后必须实际验证（方法见下）。

---

## 一、目录结构

```
docs/sql/
├── ddl.sql                     # 【全量】建表结构（29 张表，最新结构）
├── dml.sql                     # 【全量】初始化数据（公司/门店/角色/管理员/收款方式/示例业务数据）
├── 增量/                        # 【增量】升级脚本，按文件名排序依次执行
│   ├── ddl-初版.sql              #   0. 初版基线结构（等价于上线时的 ddl.sql 快照）
│   ├── dml-初版.sql              #   0. 初版基线数据
│   ├── upgrade_client_20260907.sql      # 1. 客户端三端能力（+9 表、9 处 ALTER）
│   └── upgrade_p1_pc_modules_20260910.sql # 2. P1 交付工作台/工作室设置（sys_company +2 列、biz_delivery +2 列）
└── README.md                   # 本文件
```

> `增量/` 下的「初版」两个文件是增量链的**起点**，不可删除；后续每次结构或数据变更在此目录追加新脚本。

## 二、使用方式

### 1. 新环境初始化（全量）

```bash
mysql -uroot -p < docs/sql/ddl.sql   # 建库 + 建表
mysql -uroot -p < docs/sql/dml.sql   # 初始化数据（必须在 ddl 之后）
```

默认管理员：`admin / admin123456`（`sys_user` 中为 bcrypt 密文）。

### 2. 老环境升级（增量）

```bash
# 按文件名排序依次执行，顺序：初版 → 各次 upgrade
for f in docs/sql/增量/*.sql; do
  echo ">>> $f"
  mysql -uroot -p photography < "$f"
done
```

> 增量脚本**不保证幂等**（重复执行字段已存在会报错，属预期）。执行前请备份。

## 三、维护流程（每次变更必做三步）

1. **写增量** — 在 `docs/sql/增量/` 新增 `YYYYMMDD_变更描述.sql`，只写本次变更（`ALTER TABLE` / `CREATE TABLE` / `INSERT` / `UPDATE`）。
2. **合并全量** — 把**同样的改动**合并进 `ddl.sql`（结构）或 `dml.sql`（数据），保持全量始终是最新状态。
   - 新增字段 → 直接写进 `ddl.sql` 对应表的 `CREATE TABLE`（含位置 `AFTER` 与注释）
   - 新增表 → 按业务分组插入 `ddl.sql`，并在文件头更新表数量
3. **验证一致性** — 建两个临时库分别跑「全量」与「增量链」，比对结构。

### 命名规范

- 增量文件：`YYYYMMDD_变更描述.sql`（如 `20260915_order_addon_fields.sql`）
- 描述用英文小写下划线，同一日期多次变更加后缀 `_2`

## 四、一致性验证方法（可复现）

```bash
DB=mysql-dev; U=root; P=root
cd docs/sql

# 1) 组装两套脚本（把库名换成临时库，避免污染 photography）
sed -e 's/`photography`/`_v_full`/g' ddl.sql > /tmp/v_full.sql
sed -e 's/`photography`/`_v_full`/g' dml.sql >> /tmp/v_full.sql

sed -e 's/`photography`/`_v_incr`/g' 增量/ddl-初版.sql  > /tmp/v_incr.sql
sed -e 's/`photography`/`_v_incr`/g' 增量/dml-初版.sql >> /tmp/v_incr.sql
for f in 增量/*.sql; do
  case "$f" in *初版*) continue;; esac
  sed -e 's/`photography`/`_v_incr`/g' "$f" >> /tmp/v_incr.sql
done

# 2) 导入
docker exec -i $DB mysql -u$U -p$P -e "DROP DATABASE IF EXISTS _v_full; DROP DATABASE IF EXISTS _v_incr;"
docker exec -i $DB mysql -u$U -p$P --default-character-set=utf8mb4 < /tmp/v_full.sql
docker exec -i $DB mysql -u$U -p$P --default-character-set=utf8mb4 < /tmp/v_incr.sql

# 3) 比对列 / 索引 / 表 / 数据行数（结果应为「完全一致」）
for Q in \
 "SELECT TABLE_NAME,ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,IFNULL(COLUMN_DEFAULT,'~'),EXTRA,COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME,ORDINAL_POSITION" \
 "SELECT TABLE_NAME,INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME,INDEX_NAME,SEQ_IN_INDEX" \
 "SELECT TABLE_NAME,ENGINE,TABLE_COLLATION,TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME" ; do
  docker exec $DB mysql -u$U -p$P -N -B -e "$(printf "$Q" _v_full)" > /tmp/a.txt
  docker exec $DB mysql -u$U -p$P -N -B -e "$(printf "$Q" _v_incr)" > /tmp/b.txt
  diff -q /tmp/a.txt /tmp/b.txt && echo "一致" || diff /tmp/a.txt /tmp/b.txt
done

# 4) 清理
docker exec $DB mysql -u$U -p$P -e "DROP DATABASE IF EXISTS _v_full; DROP DATABASE IF EXISTS _v_incr;"
```

## 五、通用约定

1. 每张业务表固定含：`created_by` / `created_at` / `updated_by` / `updated_at` / `deleted`（软删除）
2. 每张业务表均含 `company_id`（SaaS 多租户）
3. 所有字段带中文 `COMMENT`
4. 主键统一 `id BIGINT AUTO_INCREMENT`，软删除用 `deleted BIGINT DEFAULT 0`
5. `ddl.sql` 中每张表前保留一行中文业务说明

## 六、注意事项 / 待办

- **⚠️ 不要用 SQL 格式化工具重排 `ddl.sql`**：`USE \`photography\`;` 是 mysql **客户端命令**，格式化器会把它拆成 `USE` 与库名两行，导入时报 `ERROR: USE must be followed by a database name`，整库初始化直接失败（`CREATE DATABASE IF NOT EXISTS ...;` 同理会被拆）。若被格式化，务必先把这两条语句还原成单行再提交。表内列定义的对齐美化（含 `decimal(5, 2)` 这类写法）不影响执行，可保留。
- **字符集**：库表实际使用 `utf8`（MySQL 8 中即 `utf8mb3`），脚本内统一写 `CHARSET=utf8`；`.env` 的 `MYSQL_CHARSET=utf8mb4` 仅作用于容器建库默认值，与脚本显式声明不冲突，但**如需升级 utf8mb4（支持 emoji）必须单独写增量并转换存量数据**，不要只改脚本，否则全量与老库会不一致。
- `ddl.sql` 由真实库结构导出（`mysqldump --no-data`）后重排生成，因此字段写法为 MySQL 8 规范形式（如 `bigint` 而非 `bigint(20)`），已剥离生产数据的 `AUTO_INCREMENT` 残留值。

---

## 当前状态

| 项 | 值 |
|---|---|
| 全量表数 | 29 |
| 增量次数 | 2（`upgrade_client_20260907`、`upgrade_p1_pc_modules_20260910`） |
| 最近校验 | 2026-09-10 ✅ 列 575 行 / 索引 175 行 / 表 29 行，全部零差异 |
