#!/usr/bin/env bash
# 建立小規模測試資料（不碰真實 SVN 資料）
# 測資：2 個 repo、htpasswd、authz，以及一份可寫的 SVNAdmin data（含測試 admin 角色）
set -euo pipefail
cd "$(dirname "$0")/.."

export SVN_ROOT="$PWD/test/data/repos"
export SVN_DATA_ROOT="$PWD/test/data/appdata"

rm -rf test/data
mkdir -p "$SVN_ROOT" "$SVN_DATA_ROOT"

# 1) 灌測試倉庫與授權檔（在容器內做，svn 版本與 runtime 一致）
docker compose run --rm --entrypoint sh svn -c '
  set -e
  svnadmin create /var/svnlocal/test1
  svnadmin create /var/svnlocal/test2

  mkdir -p /tmp/wc/trunk /tmp/wc/branches
  echo "hello svn" > /tmp/wc/trunk/a.txt
  svn import -q /tmp/wc file:///var/svnlocal/test1 -m "init test1"

  htpasswd -bc /var/svnlocal/svn-auth-file testuser testpass
  htpasswd -b  /var/svnlocal/svn-auth-file admin    adminpass

  cat > /var/svnlocal/svn-access-file <<EOF
[groups]
dev = testuser

[/]
admin = rw

[test1:/]
@dev = rw
EOF

  # 讓容器內 www-data 與 host 端使用者都能讀寫測試資料
  chmod -R a+rwX /var/svnlocal 2>/dev/null || true
  # 讓 host 端使用者也能刪除測試資料（Linux 上有效，Windows/macOS 會失敗，忽略）
  chown -R 1000:1000 /var/svnlocal 2>/dev/null || true
'

# 2) 準備可寫的 SVNAdmin data（config.ini / 角色指派），並給 testuser 一個 Administrator 角色
cp -a data/. "$SVN_DATA_ROOT/"
printf '\n[testuser]\nAdministrator=\n' >> "$SVN_DATA_ROOT/userroleassignments.ini"
rm -rf "$SVN_DATA_ROOT/backup"
mkdir -p "$SVN_DATA_ROOT/backup"

echo "fixtures ready"
ls -la "$SVN_ROOT" "$SVN_DATA_ROOT"
