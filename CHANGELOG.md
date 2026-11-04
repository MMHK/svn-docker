# Changelog

本專案遵循 [Keep a Changelog](https://keepachangelog.com/zh-TW/1.1.0/) 的格式，
版本號沿用 `MAJOR.MINOR.PATCH`（本專案先前為 `1.6.2`）。

## 3.0.0 - 2026-11-04

管理介面從 PHP（iF.SVNAdmin）改寫為 **Go REST API + 內嵌 Vue 3 SPA**，拆成
`svn`（Apache + `mod_dav_svn`）與 `svn-admin`（Go）兩個映像，並把管理面限制在內網。
**破壞性變更**：舊的 PHP 介面、`.php` 路徑、單容器部署與 `admin/admin` 首次啟動流程全部移除。

### Added

- `api/`：Go module（`cmd/svnadmin-api`、MVP 分層 `pkg/{model,view,presenter,transport,store,...}`），
  提供 `/svnadmin/api/v1` REST API；`serve` / `admin grant` / `hash-password` / `check` 子命令。
- `web/`：rspack 2 + Vue 3 + TypeScript + Tailwind v4 + Pinia + Vue Router；產物 embed 進 Go binary。
- 認證：`POST /session` 發 opaque session token，`Authorization: Bearer`；TTL / 閒置 /
  rotation / 撤銷 / 每人上限；不使用 cookie，因此無 CSRF。
- 首次管理員：一次性 bootstrap token（`SVN_ADMIN_BOOTSTRAP_TOKEN`，未設時啟動印在 log）
  + `POST /setup/admin`；不再有 `admin/admin`。
- authz / htpasswd / 角色檔 **lossless 行導向編輯**：保留註解、空行、CRLF、未知 section；
  `flock` + mtime/hash 檢查 + 原子 rename + 寫入前備份。
- 倉庫刪除改為移到 `data/trash/<repo>-<ts>`，保留 30 天（`SVN_ADMIN_TRASH_RETENTION`），
  需輸入倉庫名確認；預設開放（`SVN_ADMIN_REPO_DELETE_ENABLED=true`）。
- Swagger UI（`swaggo`，離線內嵌）：`/svnadmin/api/docs/index.html`，僅管理員 + 內網。
- `Dockerfile.svn` / `Dockerfile.admin` 雙映像；`docker-entrypoint.sh` 以 `PUID`/`PGID` 對齊 uid；
  `svn-admin` 只 `expose`，永不 publish。
- 內網限制：Apache `mod_remoteip` + `RemoteIPTrustedProxy` + `<Location /svnadmin> Require ip RFC1918`，
  Go 端第二層 `SVN_ADMIN_ALLOW_CIDRS` / `SVN_ADMIN_TRUSTED_PROXIES`。
- `test/verify-v2.sh`（91 項端到端）與 `test/fixtures.sh`；`ci-v3.yml`（Go / 前端 / `make layers` /
  `make swag` 漂移）。
- `NOTICE`：保留 iF.SVNAdmin（GPL v2，Manuel Freiholz）的 attribution。

### Changed

- 版號 `2.0.0` → **`3.0.0`**；映像改為 `mmhk/svn:3.0` + `mmhk/svn-admin:3.0`。
- 設定來源改以環境變數 `SVN_ADMIN_*` 為權威；`data/config.ini` 僅作三個 SVN 路徑的唯讀 fallback。
- 進程 uid/gid 預設 **`1000:1000`**（等於 `/var/svnlocal` owner），不再依賴 `www-data`(33) 與 777。
- `/svn` 行為不變（DAV + Basic + `AuthzSVNAccessFile`）；`/svnadmin` 由 Apache `mod_proxy` 轉到 Go。
- 密碼寫入維持 APR1，讀取相容 APR1 / `{SHA}` / bcrypt / crypt。
- `readme.md` 改以 v3 為主，v2 內容只留歷史摘要（完整版見 tag `v2.0.0`）。
- `build.sh` 改為建置兩個 v3 映像。

### Removed

- **PHP 管理介面**：`app/`（iF.SVNAdmin 全部原始碼）、`assets/`、`build-assets.mjs`、
  根目錄 `package.json` / `yarn.lock` / `.yarnrc.yml`、`conf/php-svnadmin.ini`、`conf/svn-real.conf`、
  PHP 版 `Dockerfile`。
- 舊 `.php` 路由（`/svnadmin/userlist.php` 等）直接 404，不做 301。
- `test/verify.sh`、`test/verify-real.sh`、`test/probe-real-repos.sh`、`test/data/`、
  `.github/workflows/ci.yml`（PHP CI）。
- 既有資料格式不變；回滾方式見 `readme.md`「備份與回滾 2.0.0」。

## 2.0.0 - 2026-11-04

iF.SVNAdmin 現代化：Composer / PSR-4、前端控制器路由、Tailwind v4 + Primer UI、
多階段映像、PHPUnit / PHPStan / CI，並裁剪實際部署用不到的功能。

### Added

- Composer 專案（`app/composer.json`）：`SvnAdmin\` PSR-4 autoload、`constants.php` /
  `functions.php` / `globals.php` / `compat.php` 以 `files` 載入，`composer.lock` 鎖定依賴。
- 前端控制器 `app/public/index.php` + `SvnAdmin\Http\{Router,Kernel,Request,Response,Session}`，
  以 `nikic/fast-route` 比對相對於應用根的路徑；舊 `.php` 路徑保留為別名（同 handler、200）。
- 18 個頁面控制器（`SvnAdmin\Controllers\*`）與寫入操作（`SvnAdmin\Actions\*`），
  涵蓋使用者、群組、倉庫、授權路徑、權限、角色、設定。
- 分層基礎建設：`Config\Config` / `ConfigFactory` / `IniFile`、`Acl\*`、`Entities\*`、
  `Contracts\*`、`Providers\*`、`I18n\*`、`View\Template`、`Support\*`。
- Tailwind v4 建置管線（`assets/app.css` + `@tailwindcss/cli`、yarn 4）與
  Primer 風格設計系統：共用 header / navigation / footer、深色模式（`data-color-mode` +
  `color_mode` cookie）、響應式版面。
- PHPUnit 11 測試（33 tests）：`Config`、`HtPasswd`（APR1 / MD5 / SHA1）、authz 解析、
  `Router`、`Template`；`phpunit.xml` 以 `vendor/autoload.php` 啟動。
- PHPStan level 5 + `phpstan-baseline.neon`（只收歷史遺留問題，新碼不得進 baseline）。
- `.github/workflows/ci.yml`：`php -l`、`composer validate --strict`、`composer install`、
  `composer test`、`composer stan`（不跑 Docker）。
- 三階段 `Dockerfile`：`assets`（node:22-alpine 產生 Tailwind / JS）→ `vendor`
  （`composer install --no-dev --optimize-autoloader`）→ `php:8.3-apache` runtime；
  runtime 不含建置工具與 dev 依賴。
- `docker-compose.yml` healthcheck、log rotation、`SVN_HTTP_PORT` 可調埠。
- 測試腳本：`test/fixtures.sh`、`test/verify.sh`（50 項端到端）、`test/verify-real.sh`
  （真實倉庫唯讀煙霧測試）、`test/probe-real-repos.sh`。
- 本 `CHANGELOG.md`。

### Changed

- 版本號 `1.6.2` → **`2.0.0`**（`app/src/constants.php`；`MAJOR_VERSION=2`、
  `MINOR_VERSION=0.0`，沿用上游「minor 內含 patch」的表示法）。
- 映像 tag `mmhk/svn-docker:8.3` → **`mmhk/svn-docker:2.0`**（`docker-compose.yml`、
  `build.sh`、`test/verify-real.sh`、`readme.md`）；`8.3` tag 保留作為回滾點。
- iF.SVNAdmin 由 host bind mount 改為**直接內嵌映像**，只有 `data/` 由 compose 掛載。
- 執行環境升到 **PHP 8.3** + Apache 2.4 + subversion 1.14，啟用
  `dav` / `dav_svn` / `authz_*` / `access_compat` / `rewrite` 等模組。
- 靜態資源改由 `/svnadmin/assets/app.css`、`/svnadmin/assets/app.js` 提供；
  templates 圖示仍走 `/svnadmin/templates`，並以 `<base href>` 修正嵌套路由的相對路徑。
- `build.sh` 預設 tag 由 `php74` 改為 `2.0`（可用 `TAG=` 覆寫）。
- 生產 `vendor/` 只含 runtime 依賴（不含 phpunit / phpstan）。

### Removed

- `app/include/`（現代化後已無內容的空目錄）。
- `app/templates/images/ui-anim_basic_16x16.gif`（jQuery UI 殘留，已無引用）。
- 功能裁剪（不再有對應類別與設定）：
  - LDAP 認證 / 目錄查詢（只留 `passwd` 與 `basic`）
  - Digest 認證（只支援 Apache Basic Auth）
  - BDB 倉庫格式（GUI 建倉庫固定 `fsfs`）
  - Repository Update / 升級 / hotcopy 等維護操作
  - `cli/` 命令列工具
  - Repository dump / load
  - `de_ES` / `es_ES` 翻譯（只留 `en_US`、`zh_CN`）
- jQuery 與舊版 JavaScript / CSS 資產（改由 Tailwind + 原生 JS）。

### Fixed

- **R1**：`bootstrap.php` 在前端控制器下 CWD 改變，`./data/...` 相對路徑失效；
  改為 `chdir(dirname(__DIR__))`。
- **R2**：`/svnadmin/templates/*` 靜態資源 404；`conf/svn.conf` 補上 Alias。
- **R3**：首次啟動 admin 檢查在 front controller 下永遠成立、可能重定向迴圈；
  改以請求路徑判斷，跳轉目標改 `settings`。
- **R4**：`Router::dispatch()` 以完整 `REQUEST_URI`（含 `/svnadmin` 前綴）比對無前綴路由表，
  導致全部 404；改為先剝除應用 base path。
- **R5**：嵌套路由（如 `/svnadmin/users/view`）頁內相對資源解析錯誤；
  模板改用 `<base href>`。
- **R8**：`Template` 命名空間化後 `new ReflectionObject` 被解析成
  `SvnAdmin\View\ReflectionObject` 而 fatal；改為全域 `\ReflectionObject`。
- PHP 8 相容性最小修補：`$plainpasswd{0}` → `$plainpasswd[0]`；
  `utf8_encode/decode()` → `mb_convert_encoding()`；`error_reporting` 排除 `E_DEPRECATED`。
