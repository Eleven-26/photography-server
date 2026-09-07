#!/usr/bin/env bash
# ============================================================
# nacos_publish.sh — 把 config/nacos/photography-server-<profile>.yaml 推送到 Nacos
#
# 用法:
#   ./scripts/nacos_publish.sh <profile> [--dry-run]
#   profile: dev | docker.dev | test | prod（即 config/nacos/ 下模板的 data_id 后缀）
#   --dry-run: 只校验模板存在与鉴权通道，不实际推送
#
# 环境变量（建议写进 .env，真实密码勿提交 git）:
#   NACOS_PUBLISH_SERVER               Nacos 地址，默认 http://127.0.0.1:8848
#   APP_NACOS_USERNAME / APP_NACOS_PASSWORD   用户名密码登录（推荐）
#   APP_NACOS_IDENTITY_KEY / APP_NACOS_IDENTITY_VALUE
#                                      服务端 identity 互信头（免登录兜底，等于管理员权限，
#                                      仅限本地 dev；容器 NACOS_AUTH_IDENTITY_KEY/VALUE 同名透传）
#   APP_NACOS_NAMESPACE / APP_NACOS_GROUP     命名空间（默认空=public）/ 分组（默认 DEFAULT_GROUP）
#
# 注意: 推送 = 用模板内容【整体覆盖】远端该 data_id；发布前确保模板已 git commit。
#       配置变更需重启服务生效（无热更）。
# ============================================================
set -euo pipefail

PROFILE="${1:?用法: nacos_publish.sh <dev|docker.dev|test|prod> [--dry-run]}"
DRY_RUN="${2:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FILE="${SCRIPT_DIR}/../config/nacos/photography-server-${PROFILE}.yaml"

if [ ! -f "$FILE" ]; then
  echo "❌ 模板不存在: $FILE"
  echo "   可用模板:"
  ls "${SCRIPT_DIR}/../config/nacos/" 2>/dev/null | sed 's/^/   - /'
  exit 1
fi

SERVER="${NACOS_PUBLISH_SERVER:-http://127.0.0.1:8848}"
GROUP="${APP_NACOS_GROUP:-DEFAULT_GROUP}"
NAMESPACE="${APP_NACOS_NAMESPACE:-}"
DATA_ID="photography-server-${PROFILE}.yaml"

# ---- 鉴权：优先用户名密码登录，未配置时退化到 identity 互信头 ----
AUTH_DESC="未配置鉴权"
AUTH_ARGS=()
if [ -n "${APP_NACOS_USERNAME:-}" ] && [ -n "${APP_NACOS_PASSWORD:-}" ]; then
  TOKEN="$(curl -s --noproxy '*' -X POST "${SERVER}/nacos/v3/auth/user/login" \
    -d "username=${APP_NACOS_USERNAME}&password=${APP_NACOS_PASSWORD}" \
    | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')"
  if [ -n "$TOKEN" ]; then
    AUTH_DESC="用户登录 token（${APP_NACOS_USERNAME}）"
    AUTH_ARGS=(--data-urlencode "accessToken=${TOKEN}")
  else
    echo "⚠️ 登录失败（检查用户名密码/管理员是否已初始化），尝试 identity 通道…"
  fi
fi
if [ ${#AUTH_ARGS[@]} -eq 0 ] && [ -n "${APP_NACOS_IDENTITY_KEY:-}" ] && [ -n "${APP_NACOS_IDENTITY_VALUE:-}" ]; then
  AUTH_DESC="identity 互信头（${APP_NACOS_IDENTITY_KEY}）"
  AUTH_ARGS=(-H "${APP_NACOS_IDENTITY_KEY}: ${APP_NACOS_IDENTITY_VALUE}")
fi
if [ ${#AUTH_ARGS[@]} -eq 0 ]; then
  echo "❌ 未配置鉴权：设置 APP_NACOS_USERNAME/PASSWORD，或 APP_NACOS_IDENTITY_KEY/VALUE"
  exit 1
fi

echo "服务地址 : ${SERVER}"
echo "data_id  : ${DATA_ID}"
echo "group    : ${GROUP}"
echo "命名空间 : ${NAMESPACE:-（public）}"
echo "模板文件 : ${FILE} ($(wc -c < "$FILE") bytes)"
echo "鉴权方式 : ${AUTH_DESC}"

if [ "$DRY_RUN" = "--dry-run" ]; then
  echo "（--dry-run 未实际推送）"
  exit 0
fi

RESP="$(curl -s --noproxy '*' -X POST "${SERVER}/nacos/v3/admin/cs/config" \
  --data-urlencode "dataId=${DATA_ID}" \
  --data-urlencode "groupName=${GROUP}" \
  ${NAMESPACE:+--data-urlencode "namespaceId=${NAMESPACE}"} \
  --data-urlencode "type=yaml" \
  --data-urlencode "content@${FILE}" \
  "${AUTH_ARGS[@]}")"

if echo "$RESP" | grep -q '"code":0'; then
  echo "✅ 推送成功: ${DATA_ID}（重启服务后生效）"
else
  echo "❌ 推送失败，响应: ${RESP}"
  exit 1
fi
