#!/usr/bin/env bash
# 對真實資料做唯讀煙霧測試（不寫入 /var/svnlocal、不動正式 auth/access 檔）
# 直接 docker run 一個臨時容器，不另外維護一份 compose 檔
set -uo pipefail
cd "$(dirname "$0")/.."

IMAGE="${IMAGE:-mmhk/svn-docker:8.3}"
REAL_DIR="$PWD/test/real"
BASE="http://127.0.0.1:18081"
FAIL=0

rm -rf "$REAL_DIR"
mkdir -p "$REAL_DIR"

docker run --rm -v "$REAL_DIR:/out" "$IMAGE" sh -ec '
  htpasswd -bc /out/svn-auth-file smoke smoke123
  printf "[groups]\n\n[/]\nsmoke = rw\n" > /out/svn-access-file
  chown -R 1000:1000 /out 2>/dev/null || true
'

docker rm -f svn-real >/dev/null 2>&1 || true
docker run -d --name svn-real --restart no \
  -p 127.0.0.1:18081:80 \
  -e TZ=Asia/Hong_Kong \
  -v /var/svnlocal:/var/svnlocal:ro \
  -v /var/www-tools/svnadmin:/var/www-tools/svnadmin \
  -v "$PWD/conf/svn-real.conf:/etc/apache2/sites-enabled/000-default.conf:ro" \
  -v "$PWD/conf/php-svnadmin.ini:/usr/local/etc/php/conf.d/zz-svnadmin.ini:ro" \
  -v "$REAL_DIR:/test-real:ro" \
  "$IMAGE" >/dev/null
sleep 6

code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
check() { if [ "$2" = "$3" ]; then echo "  OK   $1 ($3)"; else echo "  FAIL $1 (expect $2, got $3)"; FAIL=1; fi; }

echo "== 真實倉庫（唯讀） =="
for r in $(ls /var/svnlocal | head -3); do
  [ -d "/var/svnlocal/$r/db" ] || continue
  check "GET /svn/$r/ 有認證 200" 200 "$(code -u smoke:smoke123 "$BASE/svn/$r/")"
done
check "無認證應被擋（401/403）" blocked "$(code "$BASE/svn/" | sed 's/^40[13]$/blocked/')"

echo "== 真實 iF.SVNAdmin (PHP + 正式 config.ini) =="
check "登入頁 200" 200 "$(code -L "$BASE/svnadmin/")"
if curl -sL "$BASE/svnadmin/" | grep -qiE "Fatal error|Parse error"; then
  echo "  FAIL 有 PHP 致命錯誤"; FAIL=1
else
  echo "  OK   無 PHP 致命錯誤"
fi

echo "== 容器內 svn 工具對真實倉庫（不寫入） =="
if docker exec svn-real sh -c \
     'svn info --non-interactive --no-auth-cache --username smoke --password smoke123 http://127.0.0.1/svn/RL/' >/dev/null 2>&1; then
  echo "  OK   svn info RL 成功"
else
  echo "  FAIL svn info RL 失敗"; FAIL=1
fi

docker rm -f svn-real >/dev/null 2>&1 || true
rm -rf "$REAL_DIR"

echo
[ "$FAIL" = 0 ] && echo "真實資料唯讀煙霧測試通過" || echo "有項目失敗"
exit "$FAIL"
