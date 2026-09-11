#!/usr/bin/env bash
# SQL 全量 vs 增量链 一致性校验（docs/sql/README.md 第四节的可执行版本）
#
# 用法：bash docs/sql/verify_consistency.sh
#
# 做三件事：
#   1) 组装两套脚本并导入两个临时库（_v_full / _v_incr），不污染 photography；
#   2) 比对 information_schema 的 列 / 索引 / 表 三个视图；
#   3) 清理临时库。
# 任一视图出现差异会打印 diff 前 30 行。
set -u

DB=mysql-dev; U=root; P=root
SRC="$(cd "$(dirname "$0")" && pwd)"
WORK=/tmp/sqlverify
mkdir -p "$WORK"
cd "$SRC" || exit 1

# 1) 组装脚本 —— 把库名整体替换为临时库名，避免动到 photography
sed -e 's/`photography`/`_v_full`/g' ddl.sql > "$WORK/v_full.sql"
sed -e 's/`photography`/`_v_full`/g' dml.sql >> "$WORK/v_full.sql"

sed -e 's/`photography`/`_v_incr`/g' 增量/ddl-初版.sql  > "$WORK/v_incr.sql"
sed -e 's/`photography`/`_v_incr`/g' 增量/dml-初版.sql >> "$WORK/v_incr.sql"
INCR_LIST=""
for f in 增量/*.sql; do
  case "$f" in *ddl-*|*dml-*) continue;; esac   # 跳过基线文件
  sed -e 's/`photography`/`_v_incr`/g' "$f" >> "$WORK/v_incr.sql"
  INCR_LIST="$INCR_LIST $f"
done
echo "增量链:$INCR_LIST"

# 2) 导入
docker exec -i $DB mysql -u$U -p$P -e "DROP DATABASE IF EXISTS _v_full; DROP DATABASE IF EXISTS _v_incr;" 2>/dev/null
echo "--- 导入全量 ---"
docker exec -i $DB mysql -u$U -p$P --default-character-set=utf8mb4 < "$WORK/v_full.sql" 2>&1 | grep -av "Using a password"
echo "--- 导入增量链 ---"
docker exec -i $DB mysql -u$U -p$P --default-character-set=utf8mb4 < "$WORK/v_incr.sql" 2>&1 | grep -av "Using a password"
echo "--- 导入完成（上方若空白即零错误） ---"

# 3) 比对
Q_COL="SELECT TABLE_NAME,ORDINAL_POSITION,COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,IFNULL(COLUMN_DEFAULT,'~'),EXTRA,COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME,ORDINAL_POSITION"
Q_IDX="SELECT TABLE_NAME,INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME,INDEX_NAME,SEQ_IN_INDEX"
Q_TBL="SELECT TABLE_NAME,ENGINE,TABLE_COLLATION,TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA='%s' ORDER BY TABLE_NAME"

check() {
  name="$1"; q="$2"
  docker exec $DB mysql -u$U -p$P -N -B -e "$(printf "$q" _v_full)" 2>/dev/null > "$WORK/a.txt"
  docker exec $DB mysql -u$U -p$P -N -B -e "$(printf "$q" _v_incr)" 2>/dev/null > "$WORK/b.txt"
  la=$(wc -l < "$WORK/a.txt"); lb=$(wc -l < "$WORK/b.txt")
  if diff -q "$WORK/a.txt" "$WORK/b.txt" >/dev/null; then
    echo "[一致] $name  全量=$la 行 增量链=$lb 行"
  else
    echo "[不一致] $name  全量=$la 行 增量链=$lb 行，差异前 30 行："
    diff "$WORK/a.txt" "$WORK/b.txt" | head -30
  fi
}

check "列 COLUMNS" "$Q_COL"
check "索引 STATISTICS" "$Q_IDX"
check "表 TABLES" "$Q_TBL"

# 4) 清理
docker exec -i $DB mysql -u$U -p$P -e "DROP DATABASE IF EXISTS _v_full; DROP DATABASE IF EXISTS _v_incr;" 2>/dev/null
echo "--- 临时库已清理 ---"
