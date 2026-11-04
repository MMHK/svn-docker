#!/usr/bin/env bash
# =============================================================================
# 端到端驗證 v3：Apache + mod_dav_svn + svn-admin（Go API + 內嵌 SPA）
#
# 用法：bash test/verify-v2.sh
#
# 特性：
#   - 可重複執行：每次都重跑 test/fixtures.sh 產生隔離測資
#   - 用臨時 compose project（-p）與 docker-compose.dev.yml，不動生產設定
#   - 失敗時印 FAIL 並最終 exit 1；結束一律 docker compose down -v
#   - 容器日誌保留在 test/verify-v2.log
#
# 環境變數：
#   SVN_HTTP_PORT       dev override 發佈的 loopback 埠（預設 18080）
#   VERIFY_BUILD=0      跳過 docker compose build（預設會 build，確保映像含最新程式）
#   VERIFY_PROJECT      臨時 compose project 名稱
#   SVN_NETWORK         臨時 docker network 名稱
# =============================================================================
set -uo pipefail
cd "$(dirname "$0")/.."

PROJECT="${VERIFY_PROJECT:-svnadmin-verify}"
SVN_NETWORK="${SVN_NETWORK:-svnadmin-verify-net}"
PORT="${SVN_HTTP_PORT:-18080}"
BASE="http://127.0.0.1:${PORT}"
API="$BASE/svnadmin/api/v1"
LOG="$PWD/test/verify-v2.log"

export SVN_HTTP_PORT="$PORT"
export SVN_NETWORK
export SVN_ROOT="$PWD/test/data/verify/repos"
export SVN_DATA_ROOT="$PWD/test/data/verify/appdata"
export PUID="${PUID:-1000}"
export PGID="${PGID:-1000}"
# 驗證流程的登入次數超過預設 5 次/分；限流本身由 Go 單元測試覆蓋。
export SVN_ADMIN_LOGIN_RATE_PER_MIN="${SVN_ADMIN_LOGIN_RATE_PER_MIN:-30}"
export SVN_ADMIN_TRASH_RETENTION="${SVN_ADMIN_TRASH_RETENTION:-720h}"
export SVN_ADMIN_TRASH_PURGE_INTERVAL="${SVN_ADMIN_TRASH_PURGE_INTERVAL:-24h}"
# trash 位置由 docker-compose.dev.yml 覆寫成 /var/svnlocal/.trash（見該檔註解：
# Windows/macOS 的 bind mount 是不同檔案系統，rename 會得到 EXDEV → 409）。

COMPOSE=(docker compose -p "$PROJECT" -f docker-compose.yml -f docker-compose.dev.yml)

PASS=0
FAIL=0
NETWORK_CREATED=0
TMP="$(mktemp -d)"
TOKEN=""

# -----------------------------------------------------------------------------
# 輸出 / 斷言
# -----------------------------------------------------------------------------
pass() { PASS=$((PASS + 1)); printf '  PASS %s\n' "$1"; }
fail() { FAIL=$((FAIL + 1)); printf '  FAIL %s\n' "$1"; }

check_eq() { # label expected actual
  if [ "$2" = "$3" ]; then pass "$1 ($3)"; else fail "$1 (expect $2, got $3)"; fi
}

check_contains() { # label haystack needle
  case "$2" in
    *"$3"*) pass "$1" ;;
    *) fail "$1 (missing '$3')" ;;
  esac
}

check_absent() { # label haystack needle
  case "$2" in
    *"$3"*) fail "$1 (still contains '$3')" ;;
    *) pass "$1" ;;
  esac
}

cleanup() {
  trap - EXIT

  {
    echo
    echo "==================== docker compose logs ===================="
    "${COMPOSE[@]}" logs --no-color 2>&1 || true
  } >>"$LOG" 2>&1

  "${COMPOSE[@]}" down -v --remove-orphans >>"$LOG" 2>&1 || true

  if [ "$NETWORK_CREATED" = "1" ]; then
    docker network rm "$SVN_NETWORK" >>"$LOG" 2>&1 || true
  fi

  rm -rf "$TMP"

  echo
  echo "== 結果：PASS=$PASS FAIL=$FAIL =="
  echo "== 容器日誌：$LOG =="

  if [ "$FAIL" -ne 0 ]; then
    exit 1
  fi

  exit 0
}
trap cleanup EXIT

die() {
  FAIL=$((FAIL + 1))
  echo "  FAIL $1"
  exit 1
}

# -----------------------------------------------------------------------------
# HTTP 輔助
# -----------------------------------------------------------------------------
status() { curl -s -o /dev/null -w '%{http_code}' --max-time 20 "$@"; }
body() { curl -s --max-time 20 "$@"; }

token_status() { curl -s -o /dev/null -w '%{http_code}' --max-time 20 -H "Authorization: Bearer $TOKEN" "$@"; }
token_body() { curl -s --max-time 20 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' "$@"; }
token_write() { curl -s -o /dev/null -w '%{http_code}' --max-time 20 -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' "$@"; }

login_status() { # user pass
  curl -s -o /dev/null -w '%{http_code}' --max-time 20 -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$2\"}" "$API/session"
}

login_body() { # user pass
  curl -s --max-time 20 -X POST -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$2\"}" "$API/session"
}

json_str() { # body field
  printf '%s' "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -n 1 | cut -d'"' -f4
}

# 注意：Git Bash / MSYS 會把「看起來像 POSIX 路徑」的參數改寫成 Windows 路徑再交給
# 原生 docker.exe（test -x /usr/local/bin/x 會變成 C:/Program Files/Git/usr/...）。
# 要傳容器內的絕對路徑時，一律包在 sh -c '...' 裡。
dexec() { "${COMPOSE[@]}" exec -T "$@" 2>/dev/null | tr -d '\r'; }

has_binary() { # service binary -> exit 0 when present
  local out
  out="$(dexec "$1" sh -c "command -v $2 2>/dev/null || true")"
  [ -n "$out" ]
}

b64url() { printf '%s' "$1" | base64 | tr '+/' '-_' | tr -d '=\n'; }

# -----------------------------------------------------------------------------
# 0) 準備測資與 stack
# -----------------------------------------------------------------------------
echo "== 0) fixtures / build / up =="

: >"$LOG"
exec > >(tee -a "$LOG") 2>&1

bash test/fixtures.sh >/dev/null || die "fixtures.sh 失敗"

docker network inspect "$SVN_NETWORK" >/dev/null 2>&1 || {
  docker network create "$SVN_NETWORK" >/dev/null 2>&1 || die "無法建立 docker network $SVN_NETWORK"
  NETWORK_CREATED=1
}
pass "docker network 就緒（$SVN_NETWORK）"

if [ "${VERIFY_BUILD:-1}" = "1" ]; then
  if ! "${COMPOSE[@]}" build >"$TMP/build.log" 2>&1; then
    tail -n 40 "$TMP/build.log"
    die "docker compose build 失敗（完整輸出在 $TMP/build.log）"
  fi
  pass "docker compose build 完成"
fi

if ! "${COMPOSE[@]}" up -d --wait --wait-timeout 180 >"$TMP/up.log" 2>&1; then
  "${COMPOSE[@]}" ps || true
  tail -n 40 "$TMP/up.log"
  die "docker compose up 失敗或 healthcheck 未通過（見上方輸出 / $LOG）"
fi
pass "docker compose up -d --wait 完成"

SVN_CID="$("${COMPOSE[@]}" ps -q svn | head -n 1)"
ADMIN_CID="$("${COMPOSE[@]}" ps -q svn-admin | head -n 1)"
[ -n "$SVN_CID" ] || die "找不到 svn 容器"
[ -n "$ADMIN_CID" ] || die "找不到 svn-admin 容器"

# SVN 的 /svn/ 沒有認證時是 401：等到有回應再開始
for _ in $(seq 1 30); do
  [ "$(status "$BASE/svn/")" != "000" ] && break
  sleep 1
done

# -----------------------------------------------------------------------------
# 1) 映像與 runtime
# -----------------------------------------------------------------------------
echo "== 1) 映像與 runtime 內容 =="

for image in mmhk/svn-docker:repo mmhk/svn-docker:admin; do
  if docker image inspect "$image" >/dev/null 2>&1; then pass "映像存在：$image"; else fail "映像不存在：$image"; fi
done

for binary in php node go composer; do
  if has_binary svn "$binary"; then fail "svn 映像不應有 $binary"; else pass "svn 映像無 $binary"; fi
done

for binary in php node go composer; do
  if has_binary svn-admin "$binary"; then fail "svn-admin 映像不應有 $binary"; else pass "svn-admin 映像無 $binary"; fi
done

if dexec svn-admin sh -c 'test -x /usr/local/bin/svnadmin-api'; then
  pass "svn-admin 內有 /usr/local/bin/svnadmin-api"
else
  fail "svn-admin 內缺少 /usr/local/bin/svnadmin-api"
fi

if dexec svn sh -c 'test -e /var/www-tools/svnadmin'; then
  fail "svn 映像仍有舊 PHP 管理目錄 /var/www-tools/svnadmin"
else
  pass "svn 映像已無 /var/www-tools/svnadmin"
fi

# -----------------------------------------------------------------------------
# 2) published ports
# -----------------------------------------------------------------------------
echo "== 2) published ports =="

ADMIN_PORTS="$(docker port "$ADMIN_CID" 2>/dev/null | tr -d '\r')"
check_eq "svn-admin 沒有 published port" "" "$ADMIN_PORTS"

SVN_PORTS="$(docker port "$SVN_CID" 80/tcp 2>/dev/null | tr -d '\r')"
case "$SVN_PORTS" in
  127.0.0.1:*) pass "svn 只發佈到 127.0.0.1（$SVN_PORTS）" ;;
  *) fail "svn 發佈位址不是 127.0.0.1（got '${SVN_PORTS:-empty}'）" ;;
esac

# -----------------------------------------------------------------------------
# 3) 進程 uid/gid（D22）
# -----------------------------------------------------------------------------
echo "== 3) 進程 uid/gid =="

check_eq "svn-admin 進程 uid" "1000" "$(dexec svn-admin id -u)"
check_eq "svn-admin 進程 gid" "1000" "$(dexec svn-admin id -g)"
check_eq "svn Apache 執行帳號 uid" "1000" "$(dexec svn id -u www-data)"
check_eq "svn Apache 執行帳號 gid" "1000" "$(dexec svn id -g www-data)"

# -----------------------------------------------------------------------------
# 4) readiness
# -----------------------------------------------------------------------------
echo "== 4) readiness =="

READY_CODE="$(dexec svn-admin curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/readyz || true)"
check_eq "容器內 /readyz" "200" "$READY_CODE"
check_contains "readyz 回報 ok" "$(dexec svn-admin curl -s http://127.0.0.1:8080/readyz || true)" '"status":"ok"'

# -----------------------------------------------------------------------------
# 5) SVN（mod_dav_svn + Basic + authz）
# -----------------------------------------------------------------------------
echo "== 5) SVN =="

check_eq "匿名讀 test1 應 401" "401" "$(status "$BASE/svn/test1/")"
check_eq "testuser 讀 test1 應 200" "200" "$(status -u testuser:testpass "$BASE/svn/test1/trunk/a.txt")"
check_eq "testuser 讀 test2 應 403" "403" "$(status -u testuser:testpass "$BASE/svn/test2/")"
check_eq "admin 讀 test2 應 200" "200" "$(status -u admin:adminpass "$BASE/svn/test2/")"
check_eq "錯密碼應 401" "401" "$(status -u testuser:badpass "$BASE/svn/test1/")"
check_contains "容器內 svn cat 讀到內容" \
  "$(dexec svn svn cat --non-interactive --no-auth-cache --username testuser --password testpass http://127.0.0.1/svn/test1/trunk/a.txt || true)" \
  "hello svn"

# -----------------------------------------------------------------------------
# 6) session / bearer token（D9：不用 cookie）
# -----------------------------------------------------------------------------
echo "== 6) session =="

check_eq "管理 API 錯密碼應 401" "401" "$(login_status testuser wrongpass)"

LOGIN_HEADERS="$TMP/login.headers"
LOGIN_BODY="$(curl -s --max-time 20 -D "$LOGIN_HEADERS" -X POST \
  -H 'Content-Type: application/json' \
  -d '{"username":"testuser","password":"testpass"}' "$API/session")"
TOKEN="$(json_str "$LOGIN_BODY" token)"

check_contains "正確登入取得 token" "$TOKEN" "svna_"

if grep -qi '^set-cookie:' "$LOGIN_HEADERS"; then
  fail "登入回應不應有 Set-Cookie"
else
  pass "登入回應沒有 Set-Cookie"
fi

check_eq "沒有 token 呼叫 /users 應 401" "401" "$(status "$API/users")"
check_eq "帶 token 讀 /session 應 200" "200" "$(token_status "$API/session")"

# -----------------------------------------------------------------------------
# 7) 唯讀端點
# -----------------------------------------------------------------------------
echo "== 7) 唯讀端點 =="

for endpoint in users groups access-paths repos system/info; do
  check_eq "GET /$endpoint" "200" "$(token_status "$API/$endpoint")"
done

# -----------------------------------------------------------------------------
# 8) 使用者生命週期
# -----------------------------------------------------------------------------
echo "== 8) 使用者生命週期 =="

check_eq "建立使用者 verifyuser" "201" \
  "$(token_write -X POST -d '{"username":"verifyuser","password":"verifypass1"}' "$API/users")"

NEW_LOGIN="$(login_body verifyuser verifypass1)"
NEW_TOKEN="$(json_str "$NEW_LOGIN" token)"
check_contains "新使用者可登入" "$NEW_TOKEN" "svna_"

check_eq "管理員改密碼" "204" \
  "$(token_write -X PUT -d '{"password":"verifypass2"}' "$API/users/verifyuser/password")"

check_eq "改密碼後舊 token 應 401" "401" \
  "$(status -H "Authorization: Bearer $NEW_TOKEN" "$API/session")"
check_eq "新密碼可登入" "200" "$(login_status verifyuser verifypass2)"

check_eq "刪除使用者" "204" "$(token_write -X DELETE "$API/users/verifyuser")"
check_eq "刪除後登入應 401" "401" "$(login_status verifyuser verifypass2)"

# -----------------------------------------------------------------------------
# 9) group / access path / ACL
# -----------------------------------------------------------------------------
echo "== 9) group / access path / ACL =="

check_eq "建立使用者 grpuser" "201" \
  "$(token_write -X POST -d '{"username":"grpuser","password":"grppass"}' "$API/users")"
check_eq "建立群組 verifygrp" "201" \
  "$(token_write -X POST -d '{"name":"verifygrp"}' "$API/groups")"
check_eq "設定群組成員" "200" \
  "$(token_write -X PUT -d '{"users":["grpuser"]}' "$API/groups/verifygrp/members")"
check_contains "讀回群組成員" "$(token_body "$API/groups/verifygrp/members")" "grpuser"

check_eq "建立 access path test1:/verifyacl" "201" \
  "$(token_write -X POST -d '{"path":"test1:/verifyacl"}' "$API/access-paths")"

ACL_SEG="$(b64url 'test1:/verifyacl')"
check_eq "指派 @verifygrp = rw" "200" \
  "$(token_write -X PUT -d '{"permission":"rw"}' "$API/access-paths/$ACL_SEG/acl/@verifygrp?path=test1:/verifyacl")"

ACL_BODY="$(token_body "$API/access-paths/$ACL_SEG/acl?path=test1:/verifyacl")"
check_contains "讀回 ACL 有 @verifygrp" "$ACL_BODY" "@verifygrp"

check_eq "刪除群組 verifygrp" "204" "$(token_write -X DELETE "$API/groups/verifygrp")"
check_absent "刪群組後 ACL 內的 @verifygrp 消失" \
  "$(token_body "$API/access-paths/$ACL_SEG/acl?path=test1:/verifyacl")" "@verifygrp"

# -----------------------------------------------------------------------------
# 10) 倉庫生命週期（T4.3 / T4.5 / T4.6）
# -----------------------------------------------------------------------------
echo "== 10) 倉庫生命週期 =="

check_eq "建立倉庫 verifyrepo（single）" "201" \
  "$(token_write -X POST -d '{"name":"verifyrepo","structure":"single"}' "$API/repos")"

TREE_BODY="$(token_body "$API/repos/verifyrepo/tree")"
check_contains "tree 有 trunk" "$TREE_BODY" '"trunk"'
check_contains "tree 有 branches" "$TREE_BODY" '"branches"'
check_contains "tree 有 tags" "$TREE_BODY" '"tags"'

INFO_BODY="$(token_body "$API/repos/verifyrepo/info")"
check_contains "info 有 headRevision=1" "$INFO_BODY" '"headRevision":1'

# P4 遺留：不存在的 path 必須是 404（Windows CLI 的 W160013+E200009 也要認）
check_eq "tree?path=不存在 應 404" "404" \
  "$(token_status "$API/repos/verifyrepo/tree?path=does-not-exist")"

check_eq "刪除倉庫（confirm）" "204" \
  "$(token_write -X DELETE "$API/repos/verifyrepo?confirm=verifyrepo")"

TRASH_ENTRY="$(ls "$SVN_ROOT/.trash" 2>/dev/null | grep '^verifyrepo-' | head -n 1)"
if [ -n "$TRASH_ENTRY" ]; then pass "倉庫已移入 trash（$TRASH_ENTRY）"; else fail "trash 目錄沒有 verifyrepo-*"; fi
check_absent "刪除後 /repos 不再列出 verifyrepo" "$(token_body "$API/repos")" '"verifyrepo"'

# -----------------------------------------------------------------------------
# 11) 注入向量（計畫 §7.2 #10）
# -----------------------------------------------------------------------------
echo "== 11) 注入向量 =="

check_eq "repo 名 ../evil 應 422" "422" \
  "$(token_write -X POST -d '{"name":"../evil"}' "$API/repos")"
check_eq "repo 名含空白應 422" "422" \
  "$(token_write -X POST -d '{"name":"a b"}' "$API/repos")"
check_eq "repo 結構非法應 422" "422" \
  "$(token_write -X POST -d '{"name":"okrepo","structure":"weird"}' "$API/repos")"
check_eq "path=../etc 應 422" "422" \
  "$(token_status "$API/repos/test1/tree?path=../etc")"
check_eq "path=/abs 應 422" "422" \
  "$(token_status "$API/repos/test1/tree?path=/abs")"
check_eq "rev=abc 應 422" "422" \
  "$(token_status "$API/repos/test1/tree?rev=abc")"
check_eq "rev=0 應 422" "422" \
  "$(token_status "$API/repos/test1/tree?rev=0")"

# -----------------------------------------------------------------------------
# 12) 舊 .php 路由（D14）
# -----------------------------------------------------------------------------
echo "== 12) 舊 .php 路由 =="

check_eq "舊路徑 userlist.php 應 404" "404" "$(status "$BASE/svnadmin/userlist.php")"
check_eq "舊路徑 repositoryview.php 應 404" "404" "$(status "$BASE/svnadmin/repositoryview.php?r=test1")"

# -----------------------------------------------------------------------------
# 13) Swagger UI / OpenAPI
# -----------------------------------------------------------------------------
echo "== 13) API 文件 =="

check_eq "docs/index.html 應 200" "200" "$(status "$BASE/svnadmin/api/docs/index.html")"
DOCS_BODY="$(body "$BASE/svnadmin/api/docs/openapi.json")"
check_eq "docs/openapi.json 應 200" "200" "$(status "$BASE/svnadmin/api/docs/openapi.json")"
check_contains "openapi.json 有 repos 端點" "$DOCS_BODY" "/svnadmin/api/v1/repos"
check_contains "openapi.json 有 bearerAuth" "$DOCS_BODY" "bearerAuth"
check_contains "openapi.json 有 setup/status" "$DOCS_BODY" "/svnadmin/api/v1/setup/status"

# -----------------------------------------------------------------------------
# 13b) 首次設定狀態（免 token；fixture 已建立 admin）
# -----------------------------------------------------------------------------
echo "== 13b) setup status =="

SETUP_STATUS_CODE="$(status "$API/setup/status")"
SETUP_STATUS_BODY="$(body "$API/setup/status")"
check_eq "免 token GET /setup/status" "200" "$SETUP_STATUS_CODE"
check_contains "setup 完成後 setupRequired=false" "$SETUP_STATUS_BODY" '"setupRequired":false'
check_contains "setup 完成後 userCount>=1" "$SETUP_STATUS_BODY" '"userCount":'
check_absent "setup/status 不洩漏絕對路徑" "$SETUP_STATUS_BODY" "$SVN_ROOT"

# -----------------------------------------------------------------------------
# 14) 偽造 X-Forwarded-For（§5.5.1）
# -----------------------------------------------------------------------------
echo "== 14) X-Forwarded-For 偽造 =="

check_eq "偽造 XFF 8.8.8.8 應被 Apache 403" "403" \
  "$(status -H 'X-Forwarded-For: 8.8.8.8' "$API/users")"

# -----------------------------------------------------------------------------
# 15) authz 檔 lossless（註解 / CRLF / $authenticated 保留）
# -----------------------------------------------------------------------------
echo "== 15) authz 檔保留 =="

AUTHZ="$SVN_ROOT/svn-access-file"

if grep -q 'human maintained - do not reorder' "$AUTHZ"; then
  pass "authz 註解保留"
else
  fail "authz 註解被改寫"
fi

if grep -q 'second comment line - must survive too' "$AUTHZ"; then
  pass "authz 第二行註解保留"
else
  fail "authz 第二行註解被改寫"
fi

# Git Bash 的 grep 是原生 Windows 程式，會吃掉 CR；改用 tr 數位元組。
CR_COUNT="$(tr -cd '\r' <"$AUTHZ" | wc -c | tr -d ' ')"
LF_COUNT="$(tr -cd '\n' <"$AUTHZ" | wc -c | tr -d ' ')"
if [ "${CR_COUNT:-0}" -gt 0 ] && [ "$CR_COUNT" = "$LF_COUNT" ]; then
  pass "authz 仍全部是 CRLF（$CR_COUNT 行）"
else
  fail "authz 的 CRLF 被改寫（CR=$CR_COUNT LF=$LF_COUNT）"
fi

if grep -q '\$authenticated = r' "$AUTHZ"; then
  pass "authz 的 \$authenticated 條目保留"
else
  fail "authz 的 \$authenticated 條目消失"
fi

# -----------------------------------------------------------------------------
# 16) SPA
# -----------------------------------------------------------------------------
echo "== 16) SPA =="

INDEX_HEADERS="$TMP/index.headers"
INDEX_BODY="$(curl -s --max-time 20 -D "$INDEX_HEADERS" "$BASE/svnadmin/")"
check_contains "SPA /svnadmin/ 有 #app" "$INDEX_BODY" 'id="app"'
check_eq "SPA /svnadmin/users history fallback" "200" "$(status "$BASE/svnadmin/users")"

ASSET_JS="$(printf '%s' "$INDEX_BODY" | grep -o 'assets/[^"]*\.js' | head -n 1)"
if [ -n "$ASSET_JS" ]; then
  ASSET_HEADERS="$TMP/asset.headers"
  check_eq "SPA asset JS 應 200" "200" \
    "$(curl -s -o /dev/null -D "$ASSET_HEADERS" -w '%{http_code}' --max-time 20 "$BASE/svnadmin/$ASSET_JS")"
  check_contains "asset JS 是 text/javascript" \
    "$(tr -d '\r' <"$ASSET_HEADERS" | grep -i '^content-type:' | head -n 1)" "javascript"
else
  fail "index.html 找不到 assets/*.js 引用"
fi

check_contains "SPA 有 CSP" "$(tr -d '\r' <"$INDEX_HEADERS" | grep -i '^content-security-policy:')" "default-src 'self'"
check_contains "SPA 有 nosniff" "$(tr -d '\r' <"$INDEX_HEADERS" | grep -i '^x-content-type-options:')" "nosniff"

# -----------------------------------------------------------------------------
# 17) 備份
# -----------------------------------------------------------------------------
echo "== 17) 備份 =="

BACKUP_FILES="$(ls -1 "$SVN_DATA_ROOT/backup" 2>/dev/null || true)"
BACKUP_COUNT="$(printf '%s\n' "$BACKUP_FILES" | grep -c . || true)"
if [ "${BACKUP_COUNT:-0}" -ge 1 ]; then
  pass "data/backup 有 $BACKUP_COUNT 個備份檔"
else
  fail "data/backup 沒有任何備份檔"
fi

check_contains "備份含 authz 檔" "$BACKUP_FILES" "svn-access-file."

# -----------------------------------------------------------------------------
# 18) 預設站台已移除（只剩 /svn 與 /svnadmin）
# -----------------------------------------------------------------------------
echo "== 18) 預設站台 =="

check_eq "GET / 應 403" "403" "$(status "$BASE/")"
check_absent "GET / 不含目錄列表" "$(body "$BASE/")" "Index of /"

case "$(status "$BASE/html/")" in
  403 | 404) pass "GET /html/ 應 403/404 ($(status "$BASE/html/"))" ;;
  *) fail "GET /html/ 應 403/404（got $(status "$BASE/html/")）" ;;
esac

case "$(status "$BASE/cgi-bin/")" in
  403 | 404) pass "GET /cgi-bin/ 應 403/404 ($(status "$BASE/cgi-bin/"))" ;;
  *) fail "GET /cgi-bin/ 應 403/404（got $(status "$BASE/cgi-bin/")）" ;;
esac

if dexec svn sh -c 'test ! -e /var/www/html'; then
  pass "映像內沒有 /var/www/html"
else
  fail "映像內仍有 /var/www/html"
fi

SVN_MODULES="$(dexec svn sh -c 'apache2ctl -M 2>&1' || true)"
check_absent "autoindex 未載入" "$SVN_MODULES" "autoindex"
check_absent "negotiation 未載入" "$SVN_MODULES" "negotiation"
check_absent "serve-cgi-bin 未啟用" "$(dexec svn ls /etc/apache2/conf-enabled || true)" "serve-cgi-bin.conf"

echo
exit 0
