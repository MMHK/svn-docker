# Changelog

本專案遵循 [Keep a Changelog](https://keepachangelog.com/zh-TW/1.1.0/) 的格式，
版本號沿用 `MAJOR.MINOR.PATCH`（本專案先前為 `1.6.2`）。

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
