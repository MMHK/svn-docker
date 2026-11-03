# svn-docker (Apache + mod_dav_svn + iF.SVNAdmin)

把 `/var/svnlocal` 的 SVN 服務（原本是主機上的 `apache2` + `libapache2-mod-svn` + `iF.SVNAdmin`）
整包成一個 Docker 映像：**PHP 8.3 + Apache 2.4 + subversion 1.14 + iF.SVNAdmin 1.6.2（已修 PHP 8 相容）**。

- 倉庫與授權檔維持原本的路徑與格式：`/var/svnlocal`、`svn-auth-file`、`svn-access-file`
- iF.SVNAdmin 直接打進映像，只有 `data/`（設定與角色指派）留在 host
- 不需要資料庫，完全靠 SVN 的 auth / authz 檔

## 目錄

```
.
├── Dockerfile                 php:8.3-apache + libapache2-mod-svn + subversion + mbstring
├── docker-compose.yml         service: svn / image: mmhk/svn-docker:8.3 / 80:80
├── .env                       COMPOSE_PROJECT_NAME=svn
├── build.sh                   建置（可選推 registry）
├── app/                       iF.SVNAdmin 1.6.2（PHP 8 相容修補後）
├── data/                      config.ini / userroleassignments.ini / .htaccess（掛載給容器）
├── conf/
│   ├── svn.conf               Apache vhost（DAV svn + Basic + AuthzSVNAccessFile + /svnadmin）
│   ├── svn-real.conf          唯讀煙霧測試 vhost
│   └── php-svnadmin.ini       display_errors=Off、時區、session
├── readme.md
└── test/
    ├── fixtures.sh            造小倉庫 + 測試帳號 + 可寫的 data/
    ├── verify.sh              端到端驗證（含頁面全爬、GUI 寫入操作）
    ├── verify-real.sh         對真實 /var/svnlocal 唯讀煙霧測試
    └── probe-real-repos.sh    用 1.14 工具掃真實 repo
```

## 快速開始

```sh
./build.sh                     # 產生 mmhk/svn-docker:8.3
docker compose up -d           # 預設 80:80
docker compose logs -f --tail 50
```

先不搶 80 埠：

```sh
SVN_HTTP_PORT=18080 docker compose up -d
SVN_HTTP_PORT=18080 bash test/verify.sh
```

推內網 registry：

```sh
REGISTRY=docker-registry.demo2.mixmedia.com/mmhk ./build.sh
```

## 設定

`data/config.ini` 是唯一需要調整的檔案，路徑刻意與舊主機一致：

```ini
[Subversion]
SVNAuthFile=/var/svnlocal/svn-access-file

[Repositories:svnclient]
SVNParentPath=/var/svnlocal
SvnExecutable=/usr/bin/svn
SvnAdminExecutable=/usr/bin/svnadmin

[Users:passwd]
SVNUserFile=/var/svnlocal/svn-auth-file
```

容器內會掛載：

| host | container | 說明 |
| --- | --- | --- |
| `/var/svnlocal` | `/var/svnlocal` | 倉庫 + `svn-auth-file` + `svn-access-file` |
| `./data` | `/var/www-tools/svnadmin/data` | 設定、角色指派、backup（可持久化） |
| `./conf/svn.conf` | `/etc/apache2/sites-enabled/000-default.conf` | vhost |
| `./conf/php-svnadmin.ini` | `/usr/local/etc/php/conf.d/zz-svnadmin.ini` | php.ini 覆寫 |

`/var/www` 的靜態站台不在這個 stack 範圍。

## PHP 8 相容性

iF.SVNAdmin 1.6.2 是 2013 年、以 PHP 5.3 寫的程式碼。升級到 PHP 8.3 只做了三處最小修補：

| 檔案 | 修改 | 原因 |
| --- | --- | --- |
| `app/src/Support/HtPasswd.php` | `$plainpasswd{0}` → `$plainpasswd[0]` | 大括號字串位移 PHP 8.0 移除，會直接 parse error |
| `app/src/Support/globals.php` | `utf8_encode/decode()` → `mb_convert_encoding()` | UTF-8 函式 PHP 8.2 起 deprecated |
| `app/include/config.inc.php` | `error_reporting(E_ALL)` → `E_ALL & ~E_DEPRECATED` | PHP 8 的 dynamic property 等 deprecation 太吵 |

另外確認過程式碼裡**沒有** `each()`、`create_function()`、`ereg*`、`split()`、`mysql_*`、
`$HTTP_RAW_POST_DATA`、`preg_replace /e` 這些 PHP 8 移除的用法。

## 測試

```sh
# 1) 小規模、完全不碰真實資料
bash test/fixtures.sh
SVN_ROOT=$PWD/test/data/repos SVN_DATA_ROOT=$PWD/test/data/appdata \
SVN_HTTP_PORT=18080 docker compose up -d
SVN_HTTP_PORT=18080 bash test/verify.sh

# 2) 真實資料唯讀煙霧測試（臨時容器 127.0.0.1:18081，跑完自動刪）
bash test/verify-real.sh

# 3) 只掃 repo 相容性
sh test/probe-real-repos.sh
```

`test/verify.sh` 目前涵蓋：

- `svnadmin` 是否真的內嵌在映像（只有 `data` 是掛載）
- Basic 認證 401/200、authz 權限差異（testuser 讀不到 test2）
- `svn cat`（容器內 svn client 對自己）
- GUI 登入、**18 個頁面全爬**並檢查沒有 `Fatal error` / `Warning` / `Deprecated`
- GUI 寫入操作：建使用者（寫 htpasswd）、改密碼（hash 變更）、建倉庫（`svnadmin create` + 寫 authz）

## 注意事項

- 容器內 Apache 是 `www-data`(uid 33)，`/var/svnlocal` 下的 auth 檔需可被它寫入（原主機是 777）。
- `/svn/` 目錄列表維持原行為（沒有 `SVNListParentPath`，所以 403）。
- `app/data/.htaccess` 用的是 Apache 2.2 的 `deny from all`，映像已啟用 `access_compat`。
- 443 未開放，對外 HTTPS 交給上層的 nginx/letsencrypt。
- `test/data/` 是本機測試產物，已在 `.gitignore` 排除。

## License

`app/` 為 iF.SVNAdmin 1.6.2，GPL v2（見 `app/license.txt`，原作者 Manuel Freiholz）。
本目錄其餘 Docker/compose/test 檔案沿用同一授權。
