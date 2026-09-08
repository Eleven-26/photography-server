#!/usr/bin/env bash
# ============================================================
# nacos_publish.sh — 把 config/nacos/photography-server-<profile>.yaml 推送到 Nacos
#
# 用法:
#   ./scripts/nacos_publish.sh <profile> [--dry-run]
#   profile: dev | docker.dev | test | prod（即 config/nacos/ 下模板的 data_id 后缀）
#   --dry-run: 只校验模板存在与鉴权通道，不实际推送
#
# 环境变量（建议写进 .env，脚本会自动加载；真实密码勿提交 git）:
#   NACOS_PUBLISH_SERVER               Nacos 地址，默认 http://127.0.0.1:8848
#   APP_NACOS_USERNAME / APP_NACOS_PASSWORD   用户名密码登录（推荐）
#   APP_NACOS_IDENTITY_KEY / APP_NACOS_IDENTITY_VALUE
#                                      服务端 identity 互信头（免登录兜底，等于管理员权限，
#                                      仅限本地 dev；容器 NACOS_AUTH_IDENTITY_KEY/VALUE 同名透传）
#   APP_NACOS_NAMESPACE / APP_NACOS_GROUP     命名空间（默认空=public）/ 分组（默认 DEFAULT_GROUP）
#   APP_NACOS_APP_NAME                        “归属应用”元数据（默认 photography-server）
#
# 注意: 推送 = 用模板内容【整体覆盖】远端该 data_id；发布前确保模板已 git commit。

# cd /d/www/photography-server

# ./scripts/nacos_publish.sh dev --dry-run   # 先试运行（只校验不推送）
# ./scripts/nacos_publish.sh dev             # 实际推送 dev 配置
# ./scripts/nacos_publish.sh prod            # 其他环境同理：docker.dev / test / prod

# 配置变更需重启服务生效（无热更）。
set -euo pipefail

# ---- 自动加载项目根目录 .env（仅填充尚未设置的变量，shell 已 export 的优先）----
# bash 不会像 docker compose 那样自动读 .env，这里手动加载一次，
# 让鉴权/地址变量只需维护在 .env 一处（backend 容器与推送脚本共用同名变量）。
trim() { local s="$1"; s="${s#"${s%%[![:space:]]*}"}"; s="${s%"${s##*[![:space:]]}"}"; printf '%s' "$s"; }
ENV_FILE="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}/../.env"
if [ -f "$ENV_FILE" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}" # 兼容 CRLF 行尾（Windows 编辑过的 .env）
    [ -z "$line" ] && continue
    case "$line" in \#*) continue ;; esac
    k="${line%%=*}"; v="${line#*=}"
    case "$line" in *=*) ;; *) continue ;; esac
    k="$(trim "$k")"; [ -z "$k" ] && continue
    case "$v" in
      \"*\") v="${v#\"}"; v="${v%\"}" ;;   # 双引号：内部 # 是字面量（与 compose 行为一致）
      \'*\') v="${v#\'}"; v="${v%\'}" ;;
      *) v="${v%%\#*}" ;;                  # 裸值：行内 # 视为注释
    esac
    v="$(trim "$v")"
    if [ -z "${!k:-}" ]; then export "$k=$v"; LOADED_KEYS="${LOADED_KEYS:-}${k} "; fi
  done < "$ENV_FILE"
  # dry-run 时输出从 .env 加载到的变量名（仅 key 不含值），便于排查 .env 编码/拼写问题
  if [ "${2:-}" = "--dry-run" ] || [ "${1:-}" = "--dry-run" ]; then
    if [ -n "${LOADED_KEYS:-}" ]; then
      echo "（.env 已加载变量: ${LOADED_KEYS% }）"
    else
      echo "（⚠️ .env 存在但未解析到任何变量——检查文件编码是否为 UTF-8（非 UTF-16/BOM）、变量名拼写、以及是否以 KEY=VALUE 格式书写）"
    fi
  fi
fi

PROFILE="${1:?用法: nacos_publish.sh <dev|docker.dev|test|prod> [--dry-run]}"
DRY_RUN="${2:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FILE="${SCRIPT_DIR}/../config/nacos/photography-server-${PROFILE}.yaml"

# Windows 原生 curl（C:\Windows\System32\curl.exe，Git Bash 下常见 PATH 命中它）
# 不认 MSYS 路径（/d/...），读文件参数会报 "error encountered when reading a file"。
# 用 cygpath 转为 Windows 混合路径（D:/...）再传给 curl；cygpath 为 Git Bash 自带。
CURL_FILE="$FILE"
if command -v cygpath >/dev/null 2>&1; then
  case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) CURL_FILE="$(cygpath -m "$FILE")" ;;
  esac
fi

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
APP_NAME="${APP_NACOS_APP_NAME:-photography-server}"   # Nacos 配置“归属应用”字段

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
echo "归属应用 : ${APP_NAME}"
echo "模板文件 : ${FILE} ($(wc -c < "$FILE") bytes)"
echo "鉴权方式 : ${AUTH_DESC}"

if [ "$DRY_RUN" = "--dry-run" ]; then
  echo "（--dry-run 未实际推送）"
  exit 0
fi

push_config() { # 推送请求；鉴权参数经 "$@" 展开（token 或 identity 头）
  curl -s --noproxy '*' -X POST "${SERVER}/nacos/v3/admin/cs/config" \
    --data-urlencode "dataId=${DATA_ID}" \
    --data-urlencode "groupName=${GROUP}" \
    ${NAMESPACE:+--data-urlencode "namespaceId=${NAMESPACE}"} \
    --data-urlencode "type=yaml" \
    --data-urlencode "appName=${APP_NAME}" \
    --data-urlencode "content@${CURL_FILE}" \
    "$@"
}

RESP="$(push_config "${AUTH_ARGS[@]}")"

# 登录 token 认证成功但无权限（如用户已建、角色未绑定 → 403 access denied）时，
# 若配置了 identity 互信头则自动降级重试（identity 是服务端互信通道，权限等同管理员）
if ! echo "$RESP" | grep -q '"code":0'; then
  if [ -n "${APP_NACOS_IDENTITY_KEY:-}" ] && [ -n "${APP_NACOS_IDENTITY_VALUE:-}" ]; then
    echo "⚠️ 当前鉴权推送被拒（${RESP}）"
    echo "   降级 identity 互信头重试…"
    RESP="$(push_config -H "${APP_NACOS_IDENTITY_KEY}: ${APP_NACOS_IDENTITY_VALUE}")"
  fi
fi

if echo "$RESP" | grep -q '"code":0'; then
  echo "✅ 推送成功: ${DATA_ID}（重启服务后生效）"
else
  echo "❌ 推送失败，响应: ${RESP}"
  exit 1
fi
