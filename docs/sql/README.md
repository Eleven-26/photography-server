# 数据库脚本规范（docs/sql）

多租户 SaaS，**单库多租户**（`company_id` 隔离）。

> **当前阶段：项目未上线。** 结构以 `ddl.sql` 为唯一准，数据以 `dml.sql` 为唯一准——
> 这两个文件合起来就是「第一版」。上线前的任何结构/数据变更**直接改这两个文件**即可。
>
> `增量/` 目录**保留使用**：其中 `ddl-初版.sql` / `dml-初版.sql` 是当前全量的**副本**，
> 代表增量链的起点。未上线期间二者必须与全量保持同步（见第三节第 2 步），
> 这样上线后可直接往 `增量/` 追加 `upgrade_*.sql`，无需重建基线。

> **核心不变量（任何阶段都成立）**：
> **全量脚本单独执行一次的结果 == `增量/` 目录下所有脚本按顺序依次执行的结果。**

---

## 一、目录结构

```
docs/sql/
├── ddl.sql                  # 【全量】建表结构（30 张表、586 列）
├── dml.sql                  # 【全量】初始化数据（公司/门店/角色/角色权限/管理员/收款方式/示例业务数据）
├── verify_consistency.sh    # 全量 vs 增量链 一致性校验（依赖本地 mysql-dev 容器）
├── 增量/
│   ├── ddl-初版.sql          # 增量链起点 = 第一版 ddl 快照（当前与 ddl.sql 完全一致）
│   ├── dml-初版.sql          # 增量链起点 = 第一版 dml 快照（当前与 dml.sql 完全一致）
│   └── upgrade_*_YYYYMMDD.sql  # 上线后逐次追加的增量（当前为空）
└── README.md                # 本文件
```

字段顺序规范见第五节第 1 条。

## 二、使用方式

### 新环境初始化（当前唯一方式）

```bash
mysql -uroot -p < docs/sql/ddl.sql   # 建库 + 建表
mysql -uroot -p < docs/sql/dml.sql   # 初始化数据（必须在 ddl 之后）
```

默认管理员：`admin / admin123456`（`sys_user` 中为 bcrypt 密文）。

### 老环境升级（上线后启用，当前无 upgrade）

上线后每积累一个 `upgrade_*.sql` 就按下列方式执行：

```bash
# 按文件名中的日期(yyyymmdd)升序执行，同日期按文件名序：初版 → 各次 upgrade
# ⚠️ 不要按 ls 字典序执行——文件名是「主题_日期」格式，字典序会把
#    upgrade_perm_* 排到 upgrade_role_* 之前（日期倒挂），先跑依赖后建表的脚本会中断整条链。
for f in $(ls docs/sql/增量/*.sql | awk '{n=$0; sub(/^.*\//,"",n); if (match(n,/[0-9]{8}\.sql$/)) d=substr(n,RSTART,8); else d="99999999"; print d"\t"$0}' | sort -k1,1 -k2,2 | cut -f2-); do
  echo ">>> $f"
  mysql -uroot -p photography < "$f"
done
```

> 增量脚本**不保证幂等**（重复执行字段已存在会报错，属预期）。执行前请备份。

## 三、维护流程

### 当前（未上线）：直接改全量 + 同步初版

1. **改全量** — 改 `ddl.sql`（结构）或 `dml.sql`（数据）。
2. **同步初版**（关键：让增量链起点始终等于当前全量，否则第四节校验会报不一致）：
   ```bash
   cp docs/sql/ddl.sql docs/sql/增量/ddl-初版.sql
   cp docs/sql/dml.sql docs/sql/增量/dml-初版.sql
   ```
3. **手工维护排版**：`ddl.sql` 的列定义按「字段名 → 类型 → 约束 → 中文注释」对齐，
   且每张表的字段名列宽按该表最长字段名统一补位。新增字段后如列宽变化，需整表重新对齐。
4. 变更后按第四节验证「全量 == 增量链」。

### 上线后：写增量 + 合并全量 + 验证一致

1. **写增量** — 在 `docs/sql/增量/` 新增 `upgrade_变更描述_YYYYMMDD.sql`，只写本次变更
   （`ALTER TABLE` / `CREATE TABLE` / `INSERT` / `UPDATE`）。**此后不再改动初版文件**。
2. **合并全量** — 把**同样的改动**合并进 `ddl.sql`（结构）或 `dml.sql`（数据），保持全量始终是最新状态。
   - 新增字段 → 写进 `ddl.sql` 对应表的 `CREATE TABLE`，并按第五节第 1 条的字段顺序规范插入到正确位置
   - 新增表 → 按业务分组插入 `ddl.sql`，并在文件头更新表数量
3. **验证一致性** — 跑 `verify_consistency.sh`，重建两个临时库分别执行「全量」与「增量链」比对。

### 命名规范（上线后）

- 增量文件：`upgrade_变更描述_YYYYMMDD.sql`（如 `upgrade_order_addon_fields_20260915.sql`）
- 描述用英文小写下划线；**执行顺序按文件名中的日期排序，而非文件名字典序**（字典序会因主题首字母不同造成日期倒挂，见上方示例）

## 四、一致性验证

```bash
bash docs/sql/verify_consistency.sh   # 跑完自动清理临时库
```

该脚本比对 `information_schema` 的 **列 / 索引 / 表** 三个视图，比对对象是：

- 左：`ddl.sql` + `dml.sql`（全量）
- 右：`增量/ddl-初版.sql` + `增量/dml-初版.sql` + 各 `upgrade_*`（按日期升序）

注意：列的比对是 `ORDER BY ... ORDINAL_POSITION`，**字段顺序属于校验范围** ——
所以任何一侧调整了字段顺序，另一侧必须同步调整，否则必然报不一致。

> **未上线期间同样要跑**：当前增量链只有初版、没有 upgrade，校验实际是拿「全量」与「初版副本」
> 逐列/逐索引/逐表对拍。**一旦改了全量却忘了同步初版，会立刻报不一致**——这正是它此刻的价值。
>
> 前置条件：本地需有名为 `mysql-dev` 的容器（Docker Desktop 运行中），脚本以 `docker exec` 连接。

## 五、通用约定

1. **字段顺序**：自上而下按业务权重排列，便于阅读、排障与对接口。
   1. `id`
   2. `company_id` / `store_id` —— 租户与门店键
   3. `code` / `name` / `title` / `username` —— 业务标识
   4. `status` / `type` / `category` / `level` 等 —— 主状态与分类
   5. `*_id` —— 关联外键
   6. `*_status` / `*_type` —— 派生状态
   7. 其余业务字段（金额、时间、内容等）
   8. `remark` / `note` / `description` / `sort` / `ext` 等 —— 低权重辅助字段
   9. **审计字段固定沉底**：`created_by` / `created_at` / `updated_by` / `updated_at` / `deleted`

   > 唯一例外：`sys_role_permission` 采用全量覆盖式保存、不设软删除，故只有 `created_at`，
   > 其仍置于最末。
2. 每张业务表固定含审计字段：`created_by` / `created_at` / `updated_by` / `updated_at` / `deleted`（软删除）
3. 每张业务表均含 `company_id`（SaaS 多租户）
4. 所有字段带中文 `COMMENT`
5. 主键统一 `id BIGINT AUTO_INCREMENT`，软删除用 `deleted BIGINT DEFAULT 0`
6. `ddl.sql` 中每张表前保留一行中文业务说明

## 六、注意事项 / 待办

- **⚠️ 不要用 SQL 格式化工具重排 `ddl.sql`**：`USE \`photography\`;` 是 mysql **客户端命令**，格式化器会把它拆成 `USE` 与库名两行，导入时报 `ERROR: USE must be followed by a database name`，整库初始化直接失败（`CREATE DATABASE IF NOT EXISTS ...;` 同理会被拆）。若被格式化，务必先把这两条语句还原成单行再提交。表内列定义的对齐美化（含 `decimal(5, 2)` 这类写法）不影响执行，可保留。
- **⚠️ 调整字段顺序时必须整行搬迁、不要重新补位**：每张表的字段名列宽是按该表最长字段名统一补位的，同表内所有列共用同一宽度，因此**重排行不会破坏对齐**；若顺手把每行都重新补位，全文每个字段行都会被 diff 命中，评审会被空白噪音淹没。同理，`(` 必须紧跟在 `CREATE TABLE` 之后、`PRIMARY KEY` / `KEY` 等约束行必须在列定义之后。
- **⚠️ 改了全量必须同步 `增量/ddl-初版.sql` / `dml-初版.sql`**：未上线期间两者是「同一份内容的两处存放」，漏同步会让 `verify_consistency.sh` 报假不一致，也会在上线时留下错误的基线。
- **字符集**：库表实际使用 `utf8`（MySQL 8 中即 `utf8mb3`），脚本内统一写 `CHARSET=utf8`；`.env` 的 `MYSQL_CHARSET=utf8mb4` 仅作用于容器建库默认值，与脚本显式声明不冲突，但**如需升级 utf8mb4（支持 emoji）必须单独写增量并转换存量数据**，不要只改脚本，否则全量与老库会不一致。
- `ddl.sql` 由真实库结构导出（`mysqldump --no-data`）后重排生成，因此字段写法为 MySQL 8 规范形式（如 `bigint` 而非 `bigint(20)`），已剥离生产数据的 `AUTO_INCREMENT` 残留值。

---

## 当前状态

| 项 | 值 |
|---|---|
| 阶段 | **未上线**（`ddl.sql` + `dml.sql` 即第一版） |
| 全量表数 | 30 |
| 全量列数 | 586 |
| 增量目录 | `增量/` 内 2 个初版（= 全量副本），**upgrade 次数 0** |
| 最近变更 | 2026-09-13 字段顺序重排（30 张表 / 586 列）：重要字段前置、审计字段沉底；`dml.sql` 的 9 条 INSERT 列清单同步重排；旧初版与 6 个历史 upgrade 一并作废，改以重排后的全量作为新初版 |
| 最近校验 | 静态校验通过 —— 586 列**逐字符零改动**（纯行搬迁）、非列行（`(` / 索引 / 约束）序列逐字节不变、`dml.sql` 每条 INSERT 值多重集不变。清空旧增量前已静态证明「旧增量链 30 张表字段集合 == `ddl.sql`」零差异。**未做真实建库验证**（当时 Docker daemon 未运行） |
