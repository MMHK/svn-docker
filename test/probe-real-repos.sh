#!/bin/sh
# 唯讀檢查：容器的 svn 1.14 工具是否讀得懂主機上 1.9 格式的倉庫
docker run --rm -v /var/svnlocal:/var/svnlocal:ro mixmedia/svn-apache:php74 sh -ec '
  n=0
  for d in /var/svnlocal/*/; do
    [ -f "$d/format" ] || continue
    fs=$(sed -n "s/^fs-type=//p" $d/db/fs-type 2>/dev/null)
    fmt=$(sed -n "s/^[0-9]*$//p" $d/db/format 2>/dev/null | head -1)
    if svnlook youngest "$d" >/tmp/y 2>/tmp/e; then
      echo "OK   $(basename $d) rev=$(cat /tmp/y)"
    else
      echo "FAIL $(basename $d): $(head -1 /tmp/e)"
    fi
    n=$((n+1))
    [ $n -ge 8 ] && break
  done
  echo "--- 檢查前 $n 個 repo 完成 ---"
  svnadmin --version --quiet
'
