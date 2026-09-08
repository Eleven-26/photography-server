#!/usr/bin/env bash
# ============================================================
# nacos_publish.sh — 把 config/nacos/photography-server-<profile>.yaml 推送到 Nacos
#
# 用法:
#   ./scripts/nacos_publish.sh <profile> [模式] [--force]
#   profile: dev | docker.dev | test | prod（即 config/nacos/ 下模板的 data_id 后缀）
#   模式（三选一，缺省=推送）:
#     （缺省）    推送：用本地模板【整体覆盖】远端该 data_id
#     --dry-run  只校验模板存在与鉴权通道，不实际推送
#     --diff     对比本地模板与远端内容（不推送），用于判断控制台是否被人改过
#     --pull     反向同步：把远端内容拉回本地模板（覆盖前自动备份 <file>.bak）
#   --force: 远端与本地不一致时强制覆盖（默认拒绝并提示，避免控制台改动被静默覆盖）
#
# 推送语义：整份覆盖，不是增量合并；同一份模板反复推送结果一致（幂等）。
# 所以本地模板与 Nacos 只能有一个真源。推荐以本地模板为准（可 git review + CI 校验）：
#   改配置 → 改 config/nacos/photography-server-<profile>.yaml → 跑本脚本 → 重启服务
# 若在控制台直接改过，请跑 --pull 拉回本地并提交 git，否则下次推送会把控制台改动覆盖掉。
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

# 示例
# ./scripts/nacos_publish.sh dev --dry-run   # 先试运行（只校验不推送）
# ./scripts/nacos_publish.sh dev             # 实际推送 dev 配置

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
  # dry-run / diff 时输出从 .env 加载到的变量名（仅 key 不含值），便于排查 .env 编码/拼写问题
  case " $* " in *" --dry-run "*|*" --diff "*|*" --pull "*)
    if [ -n "${LOADED_KEYS:-}" ]; then
      echo "（.env 已加载的 Nacos 相关变量: $(printf '%s' "${LOADED_KEYS% }" | tr ' ' '\n' | grep -E 'NACOS|CONFIG_SECRET' | tr '\n' ' ')）"
    else
      echo "（⚠️ .env 存在但未解析到任何变量——检查文件编码是否为 UTF-8（非 UTF-16/BOM）、变量名拼写、以及是否以 KEY=VALUE 格式书写）"
    fi
  esac
fi

PROFILE="${1:?用法: nacos_publish.sh <dev|docker.dev|test|prod> [--dry-run|--diff|--pull] [--force]}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FILE="${SCRIPT_DIR}/../config/nacos/photography-server-${PROFILE}.yaml"

# 模式解析（bash 中 VAR="x"VAR2="y" 会被并成一个赋值，务必分行写）
MODE="push"
FORCE="no"
shift
for arg in "$@"; do
  case "$arg" in
    --dry-run) MODE="dry-run" ;;
    --diff)    MODE="diff" ;;
    --pull)    MODE="pull" ;;
    --force)   FORCE="yes" ;;
    -h|--help) sed -n '2,30p' "$0"; exit 0 ;;
    *) echo "❌ 未知参数: $arg（支持 --dry-run / --diff / --pull / --force）"; exit 1 ;;
  esac
done

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
echo "执行模式 : ${MODE}$([ "$FORCE" = "yes" ] && echo ' + --force')"

# ---- 远端读取（--diff / --pull / 推送前一致性校验都依赖它）----
pull_config() { # GET 远端配置；鉴权参数经 "$@" 展开（token 或 identity 头）
  curl -s --noproxy '*' -G "${SERVER}/nacos/v3/admin/cs/config" \
    --data-urlencode "dataId=${DATA_ID}" \
    --data-urlencode "groupName=${GROUP}" \
    ${NAMESPACE:+--data-urlencode "namespaceId=${NAMESPACE}"} \
    "$@"
}

# 取远端正文：优先 python 精确解析 JSON（含 \n \" 转义），失败退化为 sed
remote_content() {
  local body="$1" out=""
  if command -v python >/dev/null 2>&1; then
    out="$(printf '%s' "$body" | python -c 'import sys,json
try:
    print(json.load(sys.stdin).get("data",{}).get("content",""),end="")
except Exception:
    pass' 2>/dev/null)"
  fi
  if [ -z "$out" ]; then
    out="$(printf '%s' "$body" | sed -n 's/.*"content":"\(.*\)"}$/\1/p' | sed -e 's/\\n/\n/g' -e 's/\\"/"/g')"
  fi
  printf '%s\n' "$out"
}
remote_md5() { printf '%s' "$1" | sed -n 's/.*"md5":"\([^"]*\)".*/\1/p'; }
# 行尾归一化后取 md5：本机工作区 LF、远端历史内容可能是 CRLF（curl 推送原样保存），
# 直接比 Nacos 返回的 md5 会因行尾差异永远不等，故统一去掉 \r 再比。
norm_md5() { tr -d '\r' | md5sum | awk '{print $1}'; }

LOCAL_MD5="$(norm_md5 < "$FILE")"
REMOTE_BODY=""
fetch_remote() { REMOTE_BODY="$(pull_config "${AUTH_ARGS[@]}")"; }

# ---- 模式：--diff（只看差异，不写任何东西）----
if [ "$MODE" = "diff" ]; then
  fetch_remote
  RMD5="$(remote_md5 "$REMOTE_BODY")"
  if [ -z "$RMD5" ]; then
    echo "❌ 远端不存在该配置或读取失败: ${DATA_ID}"
    echo "   响应: $(printf '%s' "$REMOTE_BODY" | head -c 200)"
    exit 1
  fi
  REMOTE_MD5="$(remote_content "$REMOTE_BODY" | norm_md5)"
  if [ "$REMOTE_MD5" = "$LOCAL_MD5" ]; then
    echo "✅ 本地模板与远端完全一致（内容一致；Nacos 原始 md5=${RMD5}）"
    exit 0
  fi
  echo "⚠️ 本地模板与远端不一致：本地 md5=${LOCAL_MD5} / 远端 md5=${REMOTE_MD5}（已忽略行尾差异）"
  # 用进程替换直接对比，不落临时文件（本机 rm 被安全策略包装，避免误伤）
  if command -v diff >/dev/null 2>&1; then
    diff -u <(tr -d '\r' < "$FILE") <(remote_content "$REMOTE_BODY" | tr -d '\r') || true
    echo "（diff -u 本地 → 远端：- 本地行 / + 远端行）"
  fi
  echo "→ 想保留控制台改动: $0 $PROFILE --pull"
  echo "→ 想以本地模板为准: $0 $PROFILE --force"
  exit 0
fi

# ---- 模式：--pull（远端 → 本地，覆盖前备份）----
if [ "$MODE" = "pull" ]; then
  fetch_remote
  RMD5="$(remote_md5 "$REMOTE_BODY")"
  if [ -z "$RMD5" ]; then
    echo "❌ 远端不存在该配置或读取失败: ${DATA_ID}（响应: $(printf '%s' "$REMOTE_BODY" | head -c 200)）"
    exit 1
  fi
  REMOTE_MD5="$(remote_content "$REMOTE_BODY" | norm_md5)"
  if [ "$REMOTE_MD5" = "$LOCAL_MD5" ]; then
    echo "✅ 与远端一致，无需拉取（内容一致；Nacos 原始 md5=${RMD5}）"
    exit 0
  fi
  cp "$FILE" "${FILE}.bak"
  remote_content "$REMOTE_BODY" > "$FILE"
  echo "✅ 已拉取远端内容覆盖本地模板（原文件备份: ${FILE}.bak）"
  echo "   远端 md5=${RMD5}；请 review 差异后提交 git"
  exit 0
fi

if [ "$MODE" = "dry-run" ]; then
  echo "（--dry-run 未实际推送）"
  exit 0
fi

# ---- 推送前一致性校验：远端已存在且与本地不同 → 拒绝覆盖 ----
fetch_remote
RMD5="$(remote_md5 "$REMOTE_BODY")"
REMOTE_MD5=""
[ -n "$RMD5" ] && REMOTE_MD5="$(remote_content "$REMOTE_BODY" | norm_md5)"
if [ -n "$REMOTE_MD5" ] && [ "$REMOTE_MD5" != "$LOCAL_MD5" ] && [ "$FORCE" != "yes" ]; then
  echo "❌ 远端已存在且内容与本地模板不一致（远端 md5=${REMOTE_MD5} / 本地 md5=${LOCAL_MD5}）"
  echo "   推送 = 整体覆盖，会丢掉远端当前内容。"
  echo "   查看差异: $0 $PROFILE --diff"
  echo "   保留远端: $0 $PROFILE --pull"
  echo "   确认覆盖: $0 $PROFILE --force"
  exit 1
fi

# 推送前校验：模板不得残留明文敏感值（dev 模板刻意保留明文 → --warn-only）
if [ "${SKIP_CHECK:-}" != "1" ] && command -v go >/dev/null 2>&1; then
  CHECK_FLAG=""
  [ "$PROFILE" = "dev" ] && CHECK_FLAG="--warn-only"
  if ! (cd "${SCRIPT_DIR}/.." && go run ./cmd/configctl check -f "$CURL_FILE" $CHECK_FLAG); then
    echo "❌ 模板含明文敏感值，已阻止推送（先跑 encrypt-file 加密，或 SKIP_CHECK=1 跳过校验）"
    exit 1
  fi
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
