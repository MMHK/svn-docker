# svn-docker 2.0.0 (Apache + mod_dav_svn + iF.SVNAdmin)

把 `/var/svnlocal` 的 SVN 服務（原本是主機上的 `apache2` + `libapache2-mod-svn` + `iF.SVNAdmin`）
整包成一個 Docker 映像：**PHP 8.3 + Apache 2.4 + subversion 1.14 + iF.SVNAdmin 2.0.0**。

- 倉庫與授權檔維持原本的路徑與格式：`/var/svnlocal`、`svn-auth-file`、`svn-access-file`
- iF.SVNAdmin 直接打進映像，只有 `data/`（設定與角色指派）留在 host
- 不需要資料庫，完全靠 SVN 的 auth / authz 檔
- 上游基底是 iF.SVNAdmin 1.6.2，本 stack 的現代化版本號為 **2.0.0**（Composer 化、前端控制器路由、
  Tailwind v4 UI、PHPUnit/PHPStan、多階段建置）

## 目錄

```
.
├── Dockerfile                 三階段建置：assets → vendor → php:8.3-apache
├── docker-compose.yml         service: svn / image: mmhk/svn-docker:8.3 / 80:80
├── .env                       COMPOSE_PROJECT_NAME=svn
├── build.sh                   建置（可選推 registry）
├── app/                       iF.SVNAdmin（PHP 8.3 現代化版）
│   ├── composer.json          PSR-4 autoload；scripts: test / stan
│   ├── composer.lock
│   ├── phpunit.xml            PHPUnit 11 設定（bootstrap vendor/autoload.php）
│   ├── phpstan.neon           PHPStan level 5 + baseline
│   ├── phpstan-baseline.neon  只收歷史遺留問題（新碼不得進 baseline）
│   ├── public/                前端控制器 index.php、.htaccess、建置後 assets
│   ├── src/                   PSR-4 原始碼
│   │   ├── Http/              Request / Response / Router / Kernel / Session
│   │   ├── Controllers/       頁面控制器（18 頁）
│   │   ├── Actions/           寫入操作（建使用者、改密碼、建倉庫…）
│   │   ├── Core/              Engine / ProviderFactory / 例外
│   │   ├── Config/            Config / ConfigFactory / IniFile
│   │   ├── Acl/ Entities/ Contracts/ Providers/ I18n/ View/ Support/
│   │   └── Legacy/compat.php  舊 FQCN → 新類別的過渡別名
│   ├── tests/                 PHPUnit 純邏輯測試（不需 Docker/Apache/SVN）
│   ├── views/                 18 個頁面模板 + 共用 header/navigation/footer
│   ├── templates/             圖示、圖片等靜態資源（`/svnadmin/templates`）
│   └── translations/          介面翻譯（en_US、zh_CN）
├── assets/app.css             Tailwind v4 輸入檔（`@source "../app/views"`）
├── build-assets.mjs           把 assets/app.js 複製到 app/public/assets
├── package.json / yarn.lock   yarn 4 + `@tailwindcss/cli` v4
├── data/                      config.ini / userroleassignments.ini（掛載給容器）
├── conf/
│   ├── svn.conf               Apache vhost（DAV svn + Basic + AuthzSVNAccessFile + /svnadmin）
│   ├── svn-real.conf          唯讀煙霧測試 vhost
│   └── php-svnadmin.ini       display_errors=Off、時區、session
├── readme.md
└── test/
    ├── fixtures.sh            造小倉庫 + 測試帳號 + 可寫的 data/
    ├── verify.sh              端到端驗證（共 50 項：37 項既有 + 13 項現代化檢查）
    ├── verify-real.sh         對真實 /var/svnlocal 唯讀煙霧測試
    └── probe-real-repos.sh    用 1.14 工具掃真實 repo
```

## 快速開始

```sh
./build.sh                     # 產生 mmhk/svn-docker:8.3（內含 Tailwind / Composer 產物）
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

## 路由表（新 → 舊）

所有請求都從 `app/public/index.php` 進入，由 `SvnAdmin\Http\Router` 比對「相對於應用根」的路徑
（應用掛在 `/svnadmin`）。舊 `.php` 路徑目前是**別名**，與新路徑回同一個 handler（都是 200）；
`Router::legacy()` 的 301 能力保留給下一個版本切換。

| 新路由 | 舊路徑 | 用途 |
| --- | --- | --- |
| `/svnadmin/` | `index.php` | 首頁 |
| `/svnadmin/login` | `login.php` | 登入（GET/POST） |
| `/svnadmin/logout` | `logout.php` | 登出 |
| `/svnadmin/users` | `userlist.php` | 使用者列表 |
| `/svnadmin/users/new` | `usercreate.php` | 建立使用者 |
| `/svnadmin/users/view` | `userview.php` | 使用者詳情 |
| `/svnadmin/users/change-password` | `userchangepass.php` | 改密碼 |
| `/svnadmin/users/group-assign` | `usergroupassign.php` | 指派群組 |
| `/svnadmin/groups` | `grouplist.php` | 群組列表 |
| `/svnadmin/groups/new` | `groupcreate.php` | 建立群組 |
| `/svnadmin/groups/view` | `groupview.php` | 群組詳情 |
| `/svnadmin/repositories` | `repositorylist.php` | 倉庫列表 |
| `/svnadmin/repositories/new` | `repositorycreate.php` | 建立倉庫 |
| `/svnadmin/repositories/view` | `repositoryview.php` | 倉庫詳情 |
| `/svnadmin/access-paths` | `accesspathslist.php` | 授權路徑列表 |
| `/svnadmin/access-paths/new` | `accesspathcreate.php` | 建立授權路徑 |
| `/svnadmin/access-paths/view` | `accesspathview.php` | 授權路徑詳情 |
| `/svnadmin/permissions` | `permissionassign.php` | 權限指派 |
| `/svnadmin/roles` | `rolelist.php` | 角色列表 |
| `/svnadmin/settings` | `settings.php` | 設定 |
| `/svnadmin/error` | `error.php` | 錯誤頁 |

查詢參數沿用舊介面：`?username=`、`?groupname=`、`?r=`、`?accesspath=test1%3A%2F`。
靜態資源是 `/svnadmin/assets/app.css`、`/svnadmin/assets/app.js`（`app/public/assets/`），
圖示與圖片仍在 `/svnadmin/templates/`。

## 前端建置（Tailwind v4 + yarn 4）

前端只依賴 Tailwind v4 CLI，不用 npm：

```sh
corepack enable
corepack prepare yarn@4.18.0 --activate   # 或依 package.json 的 packageManager
yarn install
yarn build                                # build:css + build:js
```

- `yarn build:css`：`tailwindcss -i ./assets/app.css -o ./app/public/assets/app.css --minify`
- `yarn build:js`：`node build-assets.mjs`（把 `assets/app.js` 複製到 `app/public/assets/`）
- 開發時用 `yarn watch:css` 持續重建
- Tailwind 以 `@source "../app/views"` 掃描模板；改 view 的 class 後要重新 build
- 深色模式：`<html data-color-mode="auto|light|dark">`，cookie `color_mode` 記住選擇，
  CSS 以 `@custom-variant dark` 產生 `dark:` 變體

映像建置時由 Node stage 重新產生上述兩份產物，**不使用** repo 裡提交的舊檔。

## 測試與靜態分析

```sh
# 1) PHPUnit（純邏輯：HtPasswd / IniFile / Router / Template / Config）
cd app && composer test          # 或 composer --working-dir=app test

# 2) PHPStan level 5（歷史問題收在 baseline，新碼不得加進去）
cd app && composer stan

# 3) 端到端（需要 Docker；先造測資）
bash test/fixtures.sh
SVN_HTTP_PORT=18080 docker compose up -d
SVN_HTTP_PORT=18080 bash test/verify.sh

# 4) 真實資料唯讀煙霧測試（臨時容器 127.0.0.1:18081，跑完自動刪）
bash test/verify-real.sh

# 5) 只掃 repo 相容性
sh test/probe-real-repos.sh
```

`test/verify.sh` 目前涵蓋：

- `svnadmin` 是否真的內嵌在映像（只有 `data` 是掛載）
- Basic 認證 401/200、authz 權限差異（testuser 讀不到 test2）
- `svn cat`（容器內 svn client 對自己）
- GUI 登入、**18 個舊頁面全爬**並檢查沒有 `Fatal error` / `Warning` / `Deprecated`
- GUI 寫入操作：建使用者（寫 htpasswd）、改密碼（hash 變更）、建倉庫（`svnadmin create` + 寫 authz）
- 現代化路徑：`/svnadmin/`、`users`、`repositories`、`access-paths`、`groups`、`settings` 皆 200
- 前端資源：`assets/app.css`、`assets/app.js` 皆 200；頁面不含 `jquery` 字樣
- 深色模式：帶 `Cookie: color_mode=dark` 時 HTML 有 `data-color-mode="dark"`
- 舊 `.php` 路徑（別名）仍 200

目前是 **50 項檢查、0 FAIL**；新增檢查不會移除任何既有項目。

## 功能裁剪

現代化時只保留實際部署用得到的 provider，以下功能已移除（不再有對應的類別與設定）：

| 功能 | 說明 |
| --- | --- |
| LDAP 認證 / 目錄查詢 | 只留 `passwd`（htpasswd）與 `basic` 認證 |
| Digest 認證 | 只支援 Apache Basic Auth |
| BDB 倉庫格式 | GUI 建倉庫固定 `fsfs` |
| Repository Update / 升級 | 移除 `svnadmin upgrade`、hotcopy 等維護操作 |
| `cli/` 命令列工具 | 只留 Web GUI |
| Repository dump / load | 不做匯出匯入 |
| `de_ES` / `es_ES` 翻譯 | 只留 `en_US`、`zh_CN` |

對應地，`data/config.ini` 的 `[Engine:Providers]` 只接受：

```ini
[Engine:Providers]
AuthenticationStatus=basic       # off | basic
UserViewProviderType=passwd
UserEditProviderType=passwd
GroupViewProviderType=svnauthfile
GroupEditProviderType=svnauthfile
AccessPathViewProviderType=svnauthfile
AccessPathEditProviderType=svnauthfile
RepositoryViewProviderType=svnclient
RepositoryEditProviderType=svnclient
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

## 映像與多階段建置

`Dockerfile` 分三階段，runtime 不含建置工具與 dev 依賴：

1. **assets**（`node:22-alpine`）：`corepack` 啟用 yarn 4 → `yarn install --immutable` → `yarn build`，
   只帶走 `app/public/assets/app.css`、`app.js`
2. **vendor**（`composer:2`）：`composer install --no-dev --optimize-autoloader`，
   只帶走 `vendor/`（不含 phpunit / phpstan）
3. **runtime**（`php:8.3-apache`）：apt 安裝 `libapache2-mod-svn` / `subversion` / mbstring，
   複製 `app/`、生產 `vendor/` 與建置後前端產物；`data/` 由 compose 掛載覆蓋

目前映像大小約 **726 MB**（`docker images mmhk/svn-docker:8.3`）。第二次建置可完整吃到 layer cache。

## PHP 8 相容性

iF.SVNAdmin 上游是 2013 年、以 PHP 5.3 寫的程式碼。升到 PHP 8.3 保留三處最小修補：

| 檔案 | 修改 | 原因 |
| --- | --- | --- |
| `app/src/Support/HtPasswd.php` | `$plainpasswd{0}` → `$plainpasswd[0]` | 大括號字串位移 PHP 8.0 移除，會直接 parse error |
| `app/src/Support/globals.php` | `utf8_encode/decode()` → `mb_convert_encoding()` | UTF-8 函式 PHP 8.2 起 deprecated |
| `app/src/bootstrap.php` | `error_reporting(E_ALL)` → `E_ALL & ~E_DEPRECATED` | PHP 8 的 dynamic property 等 deprecation 太吵 |

另外，`Template` 在命名空間化後原本寫成 `new ReflectionObject`（會被解析成
`SvnAdmin\View\ReflectionObject`），已改為全域 `\ReflectionObject`，物件迴圈才不會 fatal。

程式碼裡**沒有** `each()`、`create_function()`、`ereg*`、`split()`、`mysql_*`、
`$HTTP_RAW_POST_DATA`、`preg_replace /e` 這些 PHP 8 移除的用法，也沒有 `eval()`。

## CI

`.github/workflows/ci.yml` 在 push / PR 時跑（**不跑 Docker**）：

1. `php -l` 掃 `app/**/*.php`（排除 `app/vendor`）
2. `composer validate --strict`
3. `composer install`（含 dev）
4. `composer test`（PHPUnit）
5. `composer stan`（PHPStan level 5 + baseline）

映像建置留給手動 workflow / `build.sh`，CI 不需要 secrets。

## 注意事項

- 容器內 Apache 是 `www-data`(uid 33)，`/var/svnlocal` 下的 auth 檔需可被它寫入（原主機是 777）。
- `/svn/` 目錄列表維持原行為（沒有 `SVNListParentPath`，所以 403）。
- `app/data/.htaccess` 用的是 Apache 2.2 的 `deny from all`，映像已啟用 `access_compat`。
- 443 未開放，對外 HTTPS 交給上層的 nginx/letsencrypt。
- `test/data/` 是本機測試產物，已在 `.gitignore` 排除。
- `app/vendor/`、`node_modules/`、`.yarn/` 不入庫；`app/public/assets/` 的建置產物入庫，
  方便不裝 Node 的環境直接跑 PHPUnit。

## License

`app/` 為 iF.SVNAdmin（GPL v2，見 `app/license.txt`，原作者 Manuel Freiholz）。
本目錄其餘 Docker/compose/test 檔案沿用同一授權。
