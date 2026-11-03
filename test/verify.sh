#!/usr/bin/env bash
# 端到端驗證：Apache + mod_dav_svn + 內嵌 iF.SVNAdmin (PHP 8)
set -uo pipefail
cd "$(dirname "$0")/.."

export SVN_ROOT="${SVN_ROOT:-$PWD/test/data/repos}"
export SVN_DATA_ROOT="${SVN_DATA_ROOT:-$PWD/test/data/appdata}"
PORT="${SVN_HTTP_PORT:-8080}"
BASE="http://127.0.0.1:${PORT}"
INNER="http://127.0.0.1"   # 容器內 Apache 永遠是 80
FAIL=0
JAR="$(mktemp)"

code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
check() { # name expected actual
  if [ "$2" = "$3" ]; then echo "  OK   $1 ($3)"; else echo "  FAIL $1 (expect $2, got $3)"; FAIL=1; fi
}

echo "== 容器狀態 =="
docker compose ps

echo "== 映像/掛載（svnadmin 應在映像內，只有 data 是掛載） =="
docker inspect svn --format '{{range .Mounts}}{{.Destination}} {{end}}' | tee /tmp/svn-mounts.txt >/dev/null
if grep -q "/var/www-tools/svnadmin/data" /tmp/svn-mounts.txt; then
  echo "  OK   data 目錄有掛載（設定可持久化）"
else
  echo "  FAIL data 目錄沒有掛載"; FAIL=1
fi
if grep -qE "/var/www-tools/svnadmin " /tmp/svn-mounts.txt; then
  echo "  FAIL svnadmin 本體被掛載（應內嵌在映像內）"; FAIL=1
else
  echo "  OK   svnadmin 本體是內嵌的（沒有 bind mount）"
fi
docker compose exec -T svn php -r 'echo "PHP ", PHP_VERSION, "\n";' | sed 's/^/  /'

echo "== SVN (mod_dav_svn) =="
if docker compose exec -T svn sh -c 'test -f /var/svnlocal/svn-auth-file'; then
  echo "  OK   容器內看得到 /var/svnlocal/svn-auth-file"
else
  echo "  FAIL 容器內看不到 /var/svnlocal/svn-auth-file（SVN_ROOT 沒指到測試資料）"; FAIL=1
fi
check "無認證應 401"              401 "$(code "$BASE/svn/test1/")"
check "testuser 讀 test1 應 200"   200 "$(code -u testuser:testpass "$BASE/svn/test1/trunk/")"
check "testuser 讀 test2 應 403"   403 "$(code -u testuser:testpass "$BASE/svn/test2/")"
check "admin 讀 test2 應 200"      200 "$(code -u admin:adminpass "$BASE/svn/test2/")"
check "錯密碼應 401"              401 "$(code -u testuser:bad "$BASE/svn/test1/")"

echo "== svn client 讀寫（容器內對自己） =="
if docker compose exec -T svn svn cat --no-auth-cache \
     --username testuser --password testpass \
     "$INNER/svn/test1/trunk/a.txt" | grep -q "hello svn"; then
  echo "  OK   svn cat trunk/a.txt"
else
  echo "  FAIL svn cat trunk/a.txt"; FAIL=1
fi

echo "== iF.SVNAdmin (PHP) =="
check "登入頁應 200" 200 "$(code -L "$BASE/svnadmin/")"
if curl -sL "$BASE/svnadmin/" | grep -qi "iF.SVNAdmin"; then
  echo "  OK   頁面有 iF.SVNAdmin 字樣"
else
  echo "  FAIL 頁面沒有 iF.SVNAdmin 字樣"; FAIL=1
fi

echo "== 用測試帳號真的登入 GUI =="
LOGIN_CODE=$(curl -s -o /dev/null -w '%{http_code}' -c "$JAR" -b "$JAR" \
  -d "loginname=testuser&loginpass=testpass&login=Login" "$BASE/svnadmin/login.php")
check "登入 POST 應 302（轉 index）" 302 "$LOGIN_CODE"
if curl -s -b "$JAR" "$BASE/svnadmin/index.php" | grep -qiE "logout|testuser"; then
  echo "  OK   登入後 index.php 有登入態"
else
  echo "  FAIL 登入後 index.php 沒有登入態"; FAIL=1
fi

echo "== 所有頁面爬一遍（PHP 8 相容性） =="
PAGES=(
  "index.php"
  "repositorylist.php"
  "repositorycreate.php"
  "repositoryview.php?r=test1"
  "userlist.php"
  "usercreate.php"
  "userview.php?username=testuser"
  "userchangepass.php?username=testuser"
  "grouplist.php"
  "groupcreate.php"
  "groupview.php?groupname=dev"
  "accesspathslist.php"
  "accesspathcreate.php"
  "accesspathview.php?accesspath=test1%3A%2F"
  "permissionassign.php"
  "usergroupassign.php"
  "rolelist.php"
  "settings.php"
)
for p in "${PAGES[@]}"; do
  body="$(curl -s -b "$JAR" "$BASE/svnadmin/$p")"
  st="$(curl -s -o /dev/null -w '%{http_code}' -b "$JAR" "$BASE/svnadmin/$p")"
  bad="$(printf '%s' "$body" | grep -oE 'Fatal error|Parse error|Uncaught|Warning: |Deprecated: ' | head -1)"
  if [ "$st" = "200" ] && [ -z "$bad" ]; then
    echo "  OK   $p"
  else
    echo "  FAIL $p (status=$st${bad:+, $bad})"; FAIL=1
    printf '%s' "$body" | grep -E 'Fatal error|Parse error|Uncaught|Warning: ' | head -3 | sed 's/^/       /'
  fi
done

echo "== 寫入操作：建使用者 / 改密碼 / 建倉庫 =="
if grep -q "^testuser:" "$SVN_ROOT/svn-auth-file"; then echo "  OK   htpasswd 初始內容正確"; else echo "  FAIL htpasswd 初始內容異常"; FAIL=1; fi

curl -s -o /dev/null -b "$JAR" \
  -d "username=apiuser&password=apipass123&password2=apipass123&create=Create" \
  "$BASE/svnadmin/usercreate.php"
if grep -q "^apiuser:" "$SVN_ROOT/svn-auth-file"; then
  echo "  OK   GUI 建使用者寫進 svn-auth-file"
else
  echo "  FAIL GUI 建使用者沒寫進 svn-auth-file"; FAIL=1
fi

old_hash="$(grep '^apiuser:' "$SVN_ROOT/svn-auth-file" | cut -d: -f2)"
curl -s -o /dev/null -b "$JAR" \
  -d "username=apiuser&password=newpass123&password2=newpass123&changepass=Save" \
  "$BASE/svnadmin/userchangepass.php"
new_hash="$(grep '^apiuser:' "$SVN_ROOT/svn-auth-file" | cut -d: -f2)"
if [ -n "$new_hash" ] && [ "$old_hash" != "$new_hash" ]; then
  echo "  OK   GUI 改密碼已更新 hash"
else
  echo "  FAIL GUI 改密碼沒有更新 hash"; FAIL=1
fi

curl -s -o /dev/null -b "$JAR" \
  -d "pi=0&reponame=guitest&repotype=fsfs&accesspathcreate=1&repostructuretype=&projectname=&create=Create" \
  "$BASE/svnadmin/repositorycreate.php"
if [ -f "$SVN_ROOT/guitest/format" ]; then
  echo "  OK   GUI 建倉庫（svnadmin create 成功）"
else
  echo "  FAIL GUI 建倉庫失敗"; FAIL=1
fi
if grep -q "\[guitest:/\]" "$SVN_ROOT/svn-access-file"; then
  echo "  OK   GUI 建倉庫同時寫入 authz"
else
  echo "  FAIL GUI 沒寫入 authz 條目"; FAIL=1
fi

echo "== bind mount =="
if [ -f "$SVN_ROOT/test1/format" ]; then echo "  OK   repos 寫在 bind mount 上"; else echo "  FAIL repos 沒寫到 bind mount"; FAIL=1; fi

rm -f "$JAR" /tmp/svn-mounts.txt
echo
[ "$FAIL" = 0 ] && echo "全部通過" || echo "有項目失敗"
exit "$FAIL"
