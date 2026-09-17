#!/usr/bin/env bash
# ============================================================================
# nacos_init.sh —— Nacos 一次性初始化容器入口（docker compose 的 nacos-init 服务）
#
# 【为什么存在】Nacos 有"数据级依赖"：光起容器不够，还必须
#   ① 初始化管理员（Nacos 3.x 起用户表为空，不再预置 nacos/nacos）；
#   ② 发布业务配置 data_id，否则 backend 拉配置失败直接 fail-fast。
#   这两件事 compose 的 depends_on 表达不了，故用一个【跑完即退】的一次性容器承担，
#   backend 用 `depends_on: { nacos-init: { condition: service_completed_successfully } }` 等它。
#
# 【幂等性】`docker compose up -d` 每次都会重放本容器，故三步全部按"已达成即跳过"实现：
#   ① 先试登录，登录成功说明管理员已初始化 → 跳过初始化；
#   ② 远端配置与本地模板一致 → 跳过推送；
#   ③ 远端不存在 → 推送（首次部署）。
#   任何一步"已达成"都以退出码 0 结束，绝不阻塞 backend 启动。
#
# 【不一致时的策略】远端已存在但与本地模板不同（例如有人在控制台改过）：
#   默认【保留远端、打印警告、退出 0】——init 容器的目标是"让系统可启动"，
#   而不是"让远端等于模板"；此处若报错退出，会导致 backend 永远起不来。
#   确认要用本地模板覆盖远端时，设 NACOS_INIT_FORCE_PUSH=true。
#
# 【环境变量】（由 docker-compose.yml 注入）
#   NACOS_SERVER             Nacos 地址，默认 http://nacos:8848
#   NACOS_ADMIN_USERNAME     管理员用户名，默认 nacos
#   NACOS_ADMIN_PASSWORD     管理员密码（必填，须与 .env 的 APP_NACOS_PASSWORD 一致）
#   NACOS_INIT_ENABLE        置 false 则本容器秒退（调试期临时关掉初始化，退出码仍为 0）
#   NACOS_INIT_FORCE_PUSH     置 true 则远端不一致时强制用本地模板覆盖
#   APP_PROFILE              dev|docker.dev|test|prod，决定 data_id 后缀
#   APP_NACOS_GROUP / APP_NACOS_NAMESPACE / APP_NACOS_DATA_ID  同 backend 口径
#   NACOS_CONFIG_DIR         配置模板目录（容器内挂载点），默认 /config
#   NACOS_INIT_TIMEOUT       等待 Nacos 就绪的最长秒数，默认 180
# ============================================================================

# 注意：不开 `set -e`——本脚本大量使用"失败即走另一分支"的探测式调用。
set -uo pipefail

NACOS_SERVER="${NACOS_SERVER:-http://nacos:8848}"
ADMIN_USER="${NACOS_ADMIN_USERNAME:-nacos}"
ADMIN_PASS="${NACOS_ADMIN_PASSWORD:-}"
INIT_ENABLE="${NACOS_INIT_ENABLE:-true}"
FORCE_PUSH="${NACOS_INIT_FORCE_PUSH:-false}"
PROFILE="${APP_PROFILE:-prod}"
GROUP="${APP_NACOS_GROUP:-DEFAULT_GROUP}"
NAMESPACE="${APP_NACOS_NAMESPACE:-}"
CONFIG_DIR="${NACOS_CONFIG_DIR:-/config}"
DATA_ID="${APP_NACOS_DATA_ID:-photography-server-${PROFILE}.yaml}"
TEMPLATE_FILE="${CONFIG_DIR}/${DATA_ID}"
WAIT_TIMEOUT="${NACOS_INIT_TIMEOUT:-180}"
APP_NAME="${APP_NACOS_APP_NAME:-photography-server}"

log()  { printf '[nacos-init] %s\n' "$*"; }
warn() { printf '[nacos-init] ⚠️  %s\n' "$*" >&2; }
die()  { printf '[nacos-init] ❌ %s\n' "$*" >&2; exit 1; }

ts() { date '+%H:%M:%S'; }

# ---------------------------------------------------------------------------
# 0. 开关与参数校验
# ---------------------------------------------------------------------------
if [ "$INIT_ENABLE" != "true" ]; then
  log "NACOS_INIT_ENABLE=${INIT_ENABLE}，跳过全部初始化（如需初始化设为 true）"
  exit 0
fi
[ -n "$ADMIN_PASS" ] || die "NACOS_ADMIN_PASSWORD 为空——它必须与 .env 的 APP_NACOS_PASSWORD 一致，否则 backend 无法登录拉配置"
command -v curl >/dev/null 2>&1 || die "容器内无 curl，无法继续"

log "server=${NACOS_SERVER} profile=${PROFILE} data_id=${DATA_ID} group=${GROUP} namespace=${NAMESPACE:-（public）}"

# ---------------------------------------------------------------------------
# 1. 等待 Nacos 就绪
#    正常情况下 compose 已用 service_healthy 保证，这里再自带一轮轮询：
#    既覆盖"healthcheck 判定偏松"的可能，也让本脚本能被单独 docker run 调试。
#    候选端点按官方鉴权白名单优先级排列（/v1/console/health/** 在 ignore.urls 内，
#    开鉴权后仍免登录；/v3/admin/** 受 nacos.core.auth.admin.enabled 管控）。
# ---------------------------------------------------------------------------
wait_nacos() {
  local deadline=$(( $(date +%s) + WAIT_TIMEOUT ))
  while [ "$(date +%s)" -lt "$deadline" ]; do
    for path in /nacos/v1/console/health/readiness \
                /nacos/v3/console/health/readiness \
                /nacos/v3/admin/core/state/readiness; do
      code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "${NACOS_SERVER}${path}" 2>/dev/null)"
      if [ "$code" = "200" ]; then
        log "Nacos 就绪（${path} → 200，耗时 $( [ -n "${START_TS:-}" ] && echo "$(( $(date +%s) - START_TS ))s" || echo '?' )）"
        return 0
      fi
    done
    sleep 3
  done
  return 1
}

START_TS=$(date +%s)
if ! wait_nacos; then
  # 就绪探测失败不再硬失败：可能是端点随版本改名。转而直接试登录，
  # 登录能成说明服务其实可用；两者都不成才算真失败。
  warn "健康端点探测超时（${WAIT_TIMEOUT}s），改为直接尝试登录…"
fi

# ---------------------------------------------------------------------------
# 2. 鉴权：先试登录（幂等判据），失败再初始化管理员
# ---------------------------------------------------------------------------
login() {
  curl -s --noproxy '*' --max-time 10 -X POST "${NACOS_SERVER}/nacos/v3/auth/user/login" \
    -d "username=${ADMIN_USER}&password=${ADMIN_PASS}" 2>/dev/null \
    | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p'
}

TOKEN=""
for i in 1 2 3; do
  TOKEN="$(login)"
  [ -n "$TOKEN" ] && break
  log "登录未成功（第 ${i}/3 次，$([ "$i" = 3 ] && echo '转初始化管理员' || echo '重试')）…"
  sleep 3
done

if [ -z "$TOKEN" ]; then
  # Nacos 3.x 空用户表：调用管理员初始化接口。
  # 该接口【仅在尚无管理员时可调】，已存在时返回 {"code":409,"message":"have admin user cannot use it."}
  # ——409 正好当幂等判据（说明管理员已存在，只是我们密码不对，后面登录会再报错）。
  log "尝试初始化管理员 ${ADMIN_USER}…"
  INIT_RESP="$(curl -s --noproxy '*' --max-time 15 -X POST "${NACOS_SERVER}/nacos/v3/auth/user/admin" \
    -d "password=${ADMIN_PASS}" 2>/dev/null)"
  case "$INIT_RESP" in
    *'"code":0'*)    log "管理员初始化成功" ;;
    *'"code":409'*)  warn "管理员已存在（409 have admin user cannot use it.）——沿用既有密码，不覆盖" ;;
    *)               warn "管理员初始化响应异常：${INIT_RESP}" ;;
  esac
  sleep 2
  TOKEN="$(login)"
fi

if [ -z "$TOKEN" ]; then
  die "登录 ${NACOS_SERVER} 失败（用户 ${ADMIN_USER}）。常见原因：① 管理员密码与 APP_NACOS_PASSWORD 不一致；② 已有管理员但密码不是本 .env 的值——请先到控制台 (宿主 :\${NACOS_CONSOLE_PORT}) 确认或重置密码。注意：本容器【不会】改写已存在管理员的密码。"
fi
log "登录成功，已取得 accessToken"

AUTH=(--data-urlencode "accessToken=${TOKEN}")

# ---------------------------------------------------------------------------
# 3. 发布业务配置（幂等：远端 == 本地模板则跳过）
# ---------------------------------------------------------------------------
if [ ! -f "$TEMPLATE_FILE" ]; then
  die "配置模板不存在：${TEMPLATE_FILE}（检查 NACOS_CONFIG_DIR 挂载与 APP_PROFILE=${PROFILE} 是否有对应模板）"
fi

norm_md5() { tr -d '\r' | md5sum | awk '{print $1}'; }
remote_content() {
  printf '%s' "$1" | sed -n 's/.*"content":"\(.*\)"}$/\1/p' | sed -e 's/\\n/\n/g' -e 's/\\"/"/g'
}

REMOTE_BODY="$(curl -s --noproxy '*' --max-time 10 -G "${NACOS_SERVER}/nacos/v3/admin/cs/config" \
  --data-urlencode "dataId=${DATA_ID}" \
  --data-urlencode "groupName=${GROUP}" \
  ${NAMESPACE:+--data-urlencode "namespaceId=${NAMESPACE}"} \
  "${AUTH[@]}" 2>/dev/null)"

REMOTE_MD5=""
if printf '%s' "$REMOTE_BODY" | grep -q '"md5"'; then
  REMOTE_MD5="$(remote_content "$REMOTE_BODY" | norm_md5)"
fi
LOCAL_MD5="$(norm_md5 < "$TEMPLATE_FILE")"

if [ -z "$REMOTE_MD5" ]; then
  log "远端无该配置，推送本地模板（${DATA_ID}, $(wc -c < "$TEMPLATE_FILE") bytes）…"
elif [ "$REMOTE_MD5" = "$LOCAL_MD5" ]; then
  log "远端配置与本地模板一致，跳过推送"
  exit 0
elif [ "$FORCE_PUSH" = "true" ]; then
  warn "远端与本地不一致且 NACOS_INIT_FORCE_PUSH=true → 强制覆盖远端"
else
  # 见文件头【不一致时的策略】：保留远端、警告、退出 0（否则 backend 永远起不来）
  warn "远端配置与本地模板不一致（远端 ${REMOTE_MD5} / 本地 ${LOCAL_MD5}）"
  warn "  已【保留远端内容】并继续启动。差异对照请在本机跑："
  warn "    ./scripts/nacos_publish.sh ${PROFILE} --diff"
  warn "  确认要用本地模板覆盖远端：设 NACOS_INIT_FORCE_PUSH=true 后重跑本容器"
  exit 0
fi

PUSH_RESP="$(curl -s --noproxy '*' --max-time 15 -X POST "${NACOS_SERVER}/nacos/v3/admin/cs/config" \
  --data-urlencode "dataId=${DATA_ID}" \
  --data-urlencode "groupName=${GROUP}" \
  ${NAMESPACE:+--data-urlencode "namespaceId=${NAMESPACE}"} \
  --data-urlencode "type=yaml" \
  --data-urlencode "appName=${APP_NAME}" \
  --data-urlencode "content@${TEMPLATE_FILE}" \
  "${AUTH[@]}" 2>/dev/null)"

if printf '%s' "$PUSH_RESP" | grep -q '"code":0'; then
  log "✅ 配置发布成功：${DATA_ID}（重启 backend 后生效）"
else
  die "配置发布失败，响应：${PUSH_RESP}"
fi
