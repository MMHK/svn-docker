#!/usr/bin/env bash
# =============================================================================
# 建立 verify-v2.sh 用的隔離測試資料（不碰真實 SVN 資料、不需要 docker）
#
# 產出（預設 test/data/verify/）：
#   repos/test1             真倉庫，trunk/a.txt 已提交
#   repos/test2             真倉庫，空（r0）
#   repos/svn-auth-file     testuser/testpass、admin/adminpass（APR1）
#   repos/svn-access-file   authz：含 # 與 ; 註解、CRLF、[groups]、[test1:/]
#   appdata/userroleassignments.ini   testuser = Administrator
#   appdata/backup/         備份目錄（verify-v2 會檢查寫入後有備份檔）
#   appdata/trash/          刪倉庫的 trash 目錄
#
# 可用環境變數覆寫：VERIFY_DATA、SVNADMIN、SVN
# =============================================================================
set -euo pipefail
cd "$(dirname "$0")/.."

ROOT="${VERIFY_DATA:-$PWD/test/data/verify}"
REPOS="$ROOT/repos"
APPDATA="$ROOT/appdata"
SVNADMIN="${SVNADMIN:-svnadmin}"
SVN="${SVN:-svn}"

rm -rf "$ROOT"
mkdir -p "$REPOS" "$APPDATA/backup" "$APPDATA/trash"

# file:// URL：Git Bash / MSYS 需要 /F:/path 這種形式
file_url() {
  local path=$1
  case "$(uname -s)" in
    MINGW* | MSYS* | CYGWIN*) printf 'file:///%s' "$(cygpath -m "$path")" ;;
    *) printf 'file://%s' "$path" ;;
  esac
}

# ---- 1) 真倉庫 --------------------------------------------------------------
if command -v "$SVNADMIN" >/dev/null 2>&1 && command -v "$SVN" >/dev/null 2>&1; then
  "$SVNADMIN" create --fs-type fsfs "$REPOS/test1" >/dev/null
  "$SVNADMIN" create --fs-type fsfs "$REPOS/test2" >/dev/null

  WC="$(mktemp -d)"
  mkdir -p "$WC/trunk" "$WC/branches" "$WC/tags"
  printf 'hello svn\n' > "$WC/trunk/a.txt"
  "$SVN" import -q --non-interactive "$WC" "$(file_url "$REPOS/test1")" -m "init test1"
  rm -rf "$WC"
  echo "repos: test1 (trunk/a.txt), test2 (empty) 由 $SVNADMIN 建立" >&2
else
  echo "WARN: 找不到 $SVNADMIN/$SVN，repos/test1 與 test2 不會有真倉庫" >&2
  mkdir -p "$REPOS/test1" "$REPOS/test2"
fi

# ---- 2) htpasswd ------------------------------------------------------------
# 固定 salt 讓 hash 可重現；有 openssl 就用它算，沒有就用同 salt 的固定值。
apr1() { # password salt
  local password=$1 salt=$2
  if command -v openssl >/dev/null 2>&1; then
    openssl passwd -apr1 -salt "$salt" "$password"
    return
  fi

  case "$salt" in
    vfy00001) printf '%s' '$apr1$vfy00001$VgXGRTIviKFdbk/bPquOb/' ;;
    vfy00002) printf '%s' '$apr1$vfy00002$YLIYE3DXom1hZJOvyMAW3/' ;;
    *) echo "no openssl and no embedded hash for salt $salt" >&2; return 1 ;;
  esac
}

{
  printf '# managed by test/fixtures.sh\n'
  printf 'testuser:%s\n' "$(apr1 testpass vfy00001)"
  printf 'admin:%s\n' "$(apr1 adminpass vfy00002)"
} > "$REPOS/svn-auth-file"

# ---- 3) authz（註解 + CRLF + $authenticated 段） ----------------------------
# verify-v2.sh 會檢查這些位元組在多次寫入後仍然保留（計畫 §5.1）。
#
# 註解只用 '#'：SVN 的 authz 解析器不認 ';' 開頭的行，會把它當成語法錯誤並
# 拒絕所有存取（已實測）。Go 的 writer 仍支援 ';'（見 pkg/store/authz 測試）。
{
  printf '%s\r\n' \
    '# human maintained - do not reorder' \
    '# second comment line - must survive too' \
    '' \
    '[groups]' \
    'dev = testuser' \
    '' \
    '[/]' \
    'admin = rw' \
    '' \
    '[test1:/]' \
    '@dev = rw' \
    '' \
    '[test1:/trunk]' \
    '# read-only for every authenticated account' \
    '$authenticated = r' \
    ''
} > "$REPOS/svn-access-file"

# ---- 4) 管理資料 ------------------------------------------------------------
printf '[testuser]\nAdministrator=\n' > "$APPDATA/userroleassignments.ini"

# config.ini 只在沒有對應環境變數時被 legacy 匯入讀取；放一份讓佈局與部署一致。
if [ -f data/config.ini ]; then
  cp data/config.ini "$APPDATA/config.ini"
fi

echo "fixtures ready: $ROOT"
ls -la "$REPOS" "$APPDATA" >&2
