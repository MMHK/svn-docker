# svn-docker 3.0.0（Apache + mod_dav_svn ｜ Go svn-admin + Vue 3 SPA）

`svn-docker` 打包一套可自架的 Subversion 服務：

- **`svn`**（`mmhk/svn-docker:repo`）：Apache 2.4 + `mod_dav_svn`，提供 `/svn`（Basic + authz），
  並把 `/svnadmin` 以 `mod_proxy` 轉給管理服務。
- **`svn-admin`**（`mmhk/svn-docker:admin`）：**Go REST API**，內嵌 **rspack + Vue 3 + Tailwind CSS**
  的 SPA，只 `expose: 8080`、永不 publish。

**沒有 PHP**：v3 不再包含 iF.SVNAdmin、Composer、模板引擎或 PHP Apache 模組。runtime 也不含
Node / Go（SPA 與 Go binary 在建置階段產生並內嵌）。

不變的部分（回滾 2.0.0 的基礎）：

- 倉庫與授權檔路徑格式：`/var/svnlocal`、`svn-auth-file`（htpasswd）、`svn-access-file`（authz INI）
- v3 **不再使用角色系統**（無 Administrator 等角色）；`userroleassignments.ini` 不再讀寫，
  舊檔僅原樣保留給 2.0.0 回滾
- `/svn` 的 DAV + Basic + `AuthzSVNAccessFile` 行為

## 架構

```text
LAN ──► nginx :80 ──┬─ /svn      → svn:80 (mod_dav_svn)                        [Basic + authz]
                    └─ /svnadmin → svn:80 (mod_remoteip → Require ip RFC1918) ─► mod_proxy ─► svn-admin:8080
                                                          │
                                                          ├─ /var/svnlocal  (RW, uid 1000:1000)
                                                          └─ /data          (RW, 備份/trash/admin)
```

- nginx 只轉發 `/svn`、`/svnadmin` 給 `svn:80`，並**覆寫** `X-Forwarded-For`；不做 IP 判定。
- `svn` 啟用 **`mod_remoteip`**：信任 nginx 來源、還原真實 client，再以
  `<Location /svnadmin>` 的 `Require ip` 限制在 RFC1918 / `::1`，最後 `ProxyPass` 到 `svn-admin:8080`。
- `svn-admin` 不 publish 埠；Go 端用 `SVN_ADMIN_ALLOW_CIDRS` + `SVN_ADMIN_TRUSTED_PROXIES`
  再做一次 CIDR 檢查（第二層防線）。
- 兩容器都以 `PUID`/`PGID`（預設 `1000:1000`）執行，對齊 host 上 `/var/svnlocal` 的 owner，
  不需要 777。

## 檔案佈局

| 路徑 | 說明 |
| --- | --- |
| `Dockerfile.svn` | `debian:bookworm-slim` + apache2 + libapache2-mod-svn + subversion + curl |
| `Dockerfile.admin` | 三階段：node:22-alpine（SPA）→ golang:1.24-alpine（靜態 binary）→ debian slim |
| `docker-entrypoint.sh` | 兩映像共用：依 `PUID`/`PGID` 對齊 uid/gid |
| `conf/svn.conf` | vhost：`/svn` + `/svnadmin`（含 `mod_remoteip`） |
| `docker-compose.yml` | `svn` + `svn-admin`，接外部網路 `demo2-net` |
| `docker-compose.dev.yml` | 本機測試才把 `svn:80` 發佈到 `127.0.0.1` |
| `.env.example` | 全部 `SVN_ADMIN_*` / `SVN_ROOT` / `PUID` 等範本 |
| `api/` | Go module（`cmd/`、`pkg/` MVP 分層、Swagger docs） |
| `web/` | rspack + Vue 3 + TypeScript + Tailwind v4，產物輸出到 `api/pkg/webui/dist` |
| `test/verify-v2.sh` | 端到端驗證（104 項）；`test/fixtures.sh` 產生隔離測資 |
| `data/` | `backup/`、`trash/`、`svnadmin-admin`（掛到 `/data`） |
| `NOTICE` | iF.SVNAdmin（GPL v2）的歷史 attribution |

`api/pkg/webui/dist` 是 rspack 的建置產物，目前**入庫**，讓沒有 Node 的環境也能
`go build` 與跑測試；`yarn build` 會覆寫它。

## 快速開始

```sh
cp .env.example .env            # 調整 SVN_ROOT / PUID / PGID / SVN_NETWORK
docker network create demo2-net # 既有 nginx 網路已存在時可略過
docker compose build
docker compose up -d
docker compose ps               # 兩個 service 應為 healthy
```

`svn` 的 healthcheck 打 `/svn/`（期待 401/403），`svn-admin` 打 `/readyz`。
`svn-admin` 永遠不 publish；要從本機直接開瀏覽器時疊加 dev 檔：

```sh
SVN_HTTP_PORT=18080 docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
# http://127.0.0.1:18080/svnadmin/
```

生產環境由既有 nginx 轉發：

```nginx
location /svn/ {
    proxy_pass http://svn:80;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Real-IP $remote_addr;
}

location /svnadmin/ {
    proxy_pass http://svn:80;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## 首次啟動：bootstrap token

2.0.0 的 `admin`/`admin` 預設後門已移除。第一次建立**唯一 admin** 的流程：

1. 帳號必須先存在於 `svn-auth-file`（htpasswd）。容器內有 `htpasswd`：
   ```sh
   docker compose exec -T svn htpasswd -b /var/svnlocal/svn-auth-file admin '你的密碼'
   ```
2. 取得一次性 token：
   - `.env` 設了 `SVN_ADMIN_BOOTSTRAP_TOKEN` 就用該值；
   - 留空時，服務啟動會隨機產生並印在 log：
     ```sh
     docker compose logs svn-admin | grep bootstrap_token
     ```
3. 呼叫 setup（需從內網存取）：
   ```sh
   curl -sS -X POST http://<host>/svnadmin/api/v1/setup/admin \
     -H 'Content-Type: application/json' \
     -d '{"username":"admin","password":"<密碼>","token":"<bootstrap-token>"}'
   ```
   成功回 `201 {"username":"admin","accountCreated":true,"initializedFiles":[...]}`；
   該帳號會寫入 `data/svnadmin-admin`（`SVN_ADMIN_ADMIN_FILE`，一行帳號名）。
4. 開 `/svnadmin/`，用該帳號 + 密碼登入。session token 存在瀏覽器
   `localStorage['svnadmin.token']`，**不使用 cookie**。
5. **只有 `data/svnadmin-admin` 記錄的帳號**能登入管理 API；其他 SVN 帳號即使密碼正確，
   登入也回 `403 not_administrator`（但仍可用 `/svn` 的 Basic 認證存取倉庫）。
   `setup/admin` 只在尚未指定 admin 時可執行，已有 admin 回 `409`。

### 轉移管理員

管理員不是資料庫欄位，而是 `data/svnadmin-admin` 這個單行檔（內容為一行帳號名，
該帳號必須已存在於 `svn-auth-file`）：

```sh
echo 'newadmin' > data/svnadmin-admin   # 或直接編輯該檔
```

- 不需重啟，下一次請求即生效；舊 admin 的 session token 立即失效。
- 尚未指定 admin（檔案缺失或空）時，`GET /setup/status` 回 `reason="no_admin"`，
  必須先完成 setup。

## 內網限制與 mod_remoteip（上線前必做）

管理介面**只允許內網**。`nginx → Apache` 之後，Apache 看到的 TCP peer 是 nginx；
若不還原真實 client，`Require ip RFC1918` 會拿 nginx 的 `172.x` 比對，結果全部放行。

1. nginx 轉發時**覆寫**來源標頭（見上方範例）。
2. `conf/svn.conf` 用 `mod_remoteip` 還原真實 client：
   ```apache
   RemoteIPHeader X-Forwarded-For
   RemoteIPTrustedProxy ${SVN_REMOTEIP_TRUSTED_PROXY}
   ```
   `SVN_REMOTEIP_TRUSTED_PROXY` 由 `docker-entrypoint.sh` 注入；
   **只可收窄成 nginx 的實際位址/網段，不可放寬到 `0.0.0.0/0`、整個 RFC1918 或公網**，
   否則任何人都能偽造 `X-Forwarded-For` 繞過 `Require ip`。
   `RemoteIPTrustedProxy` 不吃主機名；docker 動態 IP 會漂移時，請在 nginx 的 compose
   給它固定 `ipv4_address`，並同步這裡的設定。
3. `/svnadmin` 只放行 `10/8`、`172.16/12`、`192.168/16`、`::1`；Go 端再用
   `SVN_ADMIN_ALLOW_CIDRS` + `SVN_ADMIN_TRUSTED_PROXIES=svn` 檢查一次。
4. 驗證：
   ```sh
   docker compose logs svn | tail -n 20
   ```
   - 內網來源打 `/svnadmin/` → `200`
   - 公網來源 → `403`
   - 從內網偽造 `curl -H 'X-Forwarded-For: 8.8.8.8'` → `403`
   - 直打 `svn-admin:8080`（容器內）→ Go 層 `403`

**直連風險**：若 `svn:80` 被 publish 到 LAN，直連仍會經過同一組 `Require ip`（XFF 不被信任、
以 TCP peer 判定），但會繞過 nginx 的日誌與限流；因此預設不 publish，僅 dev 檔綁 loopback。
禁止 `--network host`。

**`mod_remoteip` 的實際語意（實測）**：`RemoteIPTrustedProxy` 只把**公網**位址當成 useragent，
XFF 裡的私有位址不會被採信，`%a` 會顯示 nginx 的容器 IP；存取判定仍正確（公網一律 `403`）。
若需要在 log 看到內網 client IP，把該行改成 `RemoteIPInternalProxy`（前提是 nginx 覆寫 XFF），
變更前請重跑上面的驗證。

IPv4 loopback（`127.0.0.1`）不在 `/svnadmin` allowlist（只有 `::1` 與 RFC1918），這是刻意的；
容器內自我測試請用 `http://svn/svnadmin/...`。

**不提供預設站台**：`svn` 映像移除 `autoindex` / `negotiation` 模組與 `serve-cgi-bin.conf`，
刪除 `/var/www/html`，並由 `conf/svn.conf` 的 `<Directory /var/www>` 明確拒絕。
`/`、`/html/`、`/cgi-bin/` 一律回 `403`（或 `404`），不會列出目錄；只有 `/svn` 與 `/svnadmin` 可用。

## 設定（環境變數）

v3 以環境變數為權威（`SVN_ADMIN_*`），完整範本見 `.env.example`。
`data/config.ini` 只在三個 SVN 路徑未設定時作為唯讀 fallback（方便沿用 v2 的資料夾）。

| 變數 | 預設 | 說明 |
| --- | --- | --- |
| `SVN_ROOT` | `/var/svnlocal` | host 端 SVN 根目錄，兩容器同時掛載 |
| `SVN_DATA_ROOT` | `./data` | host 端管理資料，掛到 `svn-admin` 的 `/data` |
| `PUID` / `PGID` | `1000` / `1000` | 進程 uid/gid，**必須等於 `SVN_ROOT` owner** |
| `SVN_NETWORK` | `demo2-net` | 外部 docker 網路名稱 |
| `SVN_HTTP_PORT` | `18080` | 只有疊 `docker-compose.dev.yml` 才生效（綁 127.0.0.1） |
| `SVN_REMOTEIP_TRUSTED_PROXY` | `172.16.0.0/12 192.168.0.0/16` | Apache 信任的代理來源；只可收窄 |
| `SVN_ADMIN_ADDR` | `:8080` | Go 監聽位址 |
| `SVN_ADMIN_SVN_PARENT_PATH` | `/var/svnlocal` | `SVNParentPath` |
| `SVN_ADMIN_SVN_AUTH_FILE` | `/var/svnlocal/svn-auth-file` | htpasswd |
| `SVN_ADMIN_SVN_ACCESS_FILE` | `/var/svnlocal/svn-access-file` | authz INI |
| `SVN_ADMIN_BACKUP_DIR` | `/data/backup` | 寫入前備份 |
| `SVN_ADMIN_TRASH_DIR` | `/data/trash` | 刪倉庫的回收區 |
| `SVN_ADMIN_ADMIN_FILE` | `/data/svnadmin-admin` | 唯一 admin 帳號（單行檔；改此檔即轉移管理員） |
| `SVN_ADMIN_TRASH_RETENTION` | `720h`（30 天） | trash 保留期 |
| `SVN_ADMIN_TRASH_PURGE_INTERVAL` | `24h` | trash 清除器間隔（啟動先跑一次） |
| `SVN_ADMIN_REPO_DELETE_ENABLED` | `true` | 刪倉庫開關（仍需輸入倉庫名確認） |
| `SVN_ADMIN_UID` / `SVN_ADMIN_GID` | `1000` / `1000` | 服務層記錄的進程身份 |
| `SVN_ADMIN_ENABLE_DOCS` | `true` | Swagger UI 開關（仍僅 admin + 內網） |
| `SVN_ADMIN_DEFAULT_LOCALE` | `zh_CN` | 預設語系（`zh_CN` / `en_US`） |
| `SVN_ADMIN_BOOTSTRAP_TOKEN` | 空 | 空＝啟動時隨機產生並印 log |
| `SVN_ADMIN_ALLOW_CIDRS` | `10/8,172.16/12,192.168/16` | Go 層來源限制 |
| `SVN_ADMIN_TRUSTED_PROXIES` | `svn` | 只信任 Apache 送來的 XFF |
| `SVN_ADMIN_TOKEN_TTL` | `12h` | session token 絕對上限 |
| `SVN_ADMIN_TOKEN_IDLE` | `2h` | 閒置上限 |
| `SVN_ADMIN_LOGIN_RATE_PER_MIN` | `5` | 登入限流（次/分/IP） |

權限（重要）：`SVN_ROOT` 與 `SVN_DATA_ROOT` 的 owner 必須是 `PUID:PGID`。
entrypoint **不會** chown 這兩個目錄，部署前請自行確認：

```sh
sudo chown -R 1000:1000 /var/svnlocal ./data
```

## API 文件

Swagger UI：`/svnadmin/api/docs/index.html`（`SVN_ADMIN_ENABLE_DOCS=true` 時）。
規格：`/svnadmin/api/docs/openapi.json`、`openapi.yaml`。
與其他管理端點同一組保護：必須登入（唯一 admin）且受 RFC1918 限制。
`swag` 產物在 `api/pkg/docs/`，CI 會執行 `make swag` 後比對 `git diff` 檢查漂移。

## 開發

```sh
# Go（Windows 需關掉 buildvcs：本 repo 位於 SVN working copy 內）
cd api
GOFLAGS=-buildvcs=false go build ./...
GOFLAGS=-buildvcs=false go test -race ./...
make layers     # presenter 不得 import net/http
make swag       # 重新產生 pkg/docs

# 前端（yarn 4；不要用 npm）
cd web
yarn install --immutable
yarn dev        # :5173，proxy /svnadmin/api → :8080
yarn build      # 輸出到 ../api/pkg/webui/dist
yarn typecheck
```

## 驗證

端到端驗證需要 Docker，會建立臨時 compose project、跑 `test/fixtures.sh`，
最後自動 `docker compose down -v`：

```sh
bash test/verify-v2.sh          # 104 項；本機 compose 會 publish 8085/8086，故預期 2 FAIL
```

Windows 請用 Git Bash：

```powershell
& 'C:\Program Files\Git\bin\bash.exe' test/verify-v2.sh
```

涵蓋：映像內容（無 php/node/go/composer）、published port、uid/gid、`/readyz`、
SVN Basic/authz、session token、唯讀端點、使用者/群組/ACL/倉庫生命週期、
注入向量、舊 `.php` 404、Swagger、偽造 XFF 403、authz lossless、SPA/CSP、備份。

## 映像與建置

```sh
docker build -f Dockerfile.svn   -t mmhk/svn-docker:repo .
docker build -f Dockerfile.admin -t mmhk/svn-docker:admin .
# 或一次建兩個（可選 REGISTRY=... 推送）
./build.sh
```

| 映像 | 內容 | 大小目標 |
| --- | --- | --- |
| `mmhk/svn-docker:repo` | apache2 + libapache2-mod-svn + subversion + curl | ≤ 300 MB |
| `mmhk/svn-docker:admin` | subversion CLI + Go 靜態 binary（SPA 已 embed） | ≤ 150 MB |

runtime 不含 PHP / Node / Go / Composer；`svn-admin` 以非 root 執行。

## 備份與回滾 2.0.0

v3 只寫 `svn-auth-file`（htpasswd）與 `svn-access-file`（authz INI），**不寫**
`userroleassignments.ini`（角色系統已移除，舊檔原樣保留給 2.0.0 回滾）；
2.0.0 可讀回原有資料，每次寫入前另有 `data/backup/` 備份。

回滾前先停 v3 stack 並備份 `/var/svnlocal` 與 `data/`：

```sh
docker compose down
tar czf svn-backup-$(date +%F).tgz -C /var/svnlocal . -C "$PWD/data" .
```

方式 A：改用舊映像（若 registry 仍保留 `mmhk/svn-docker:2.0`）：

```sh
docker run -d --name svn-v2 -p 18080:80 \
  -v /var/svnlocal:/var/svnlocal -v "$PWD/data:/var/www-tools/svnadmin/data" \
  -v "$PWD/old-conf/svn.conf:/etc/apache2/sites-enabled/000-default.conf:ro" \
  mmhk/svn-docker:2.0
```

方式 B：用 tag 開 worktree（推薦，conf/compose 一起回到 v2）：

```sh
git worktree add ../svn-docker-rollback v2.0.0
cd ../svn-docker-rollback
docker compose up -d
```

- v2 容器以 `www-data`(33) 執行；v3 期間新建的檔案 owner 是 1000，寫入失敗時可
  `chown -R 33:33 /var/svnlocal ./data`（僅回滾期間）。
- v3 的 `/svnadmin` 不再有 PHP，舊 `.php` 書籤在 v3 是 404（2.0.0 仍可用）。
- 回滾後如需再回 v3，記得把 owner 改回 `1000:1000`。

## 授權

v3 的程式碼（`api/`、`web/`、Dockerfile、entrypoint、`conf/`、`test/`、文件）為本專案自有實作。

`app/`（iF.SVNAdmin，GPL v2，原作者 Manuel Freiholz）在 v2.0.0 之前存在，已於 v3 移除；
v3 是行為重寫，**不含**其原始碼。GPL 授權條文與 attribution 保留在 git tag `v2.0.0`
（原 `app/license.txt`），摘要見 `NOTICE`。

---

# 附錄：v2.0.0 歷史（PHP 8.3 + iF.SVNAdmin）

2.0.0 是 iF.SVNAdmin 的現代化版本：Composer / PSR-4、前端控制器路由、Tailwind v4 + Primer UI、
多階段 `Dockerfile`、PHPUnit / PHPStan / CI，並裁剪掉 LDAP、Digest、BDB、dump/load 等功能。
它是一個 `php:8.3-apache` 單容器（`mmhk/svn-docker:2.0`），`/svn` 與 `/svnadmin` 同 vhost，
管理介面由 PHP 直接 `exec()` 呼叫 `svn` / `svnadmin`，以 `www-data`(33) 寫入 `1000:1000` 的資料，
實務上要靠 777 放寬權限。

v3 針對這些問題改成雙容器、Go API + SPA、uid 1000 對齊、內網 `mod_remoteip`；
完整的 v2 說明（含 `admin/admin` 首次啟動流程、路由表、PHP 8 相容性修補）請見
`git show v2.0.0:readme.md`。需要對照或回滾時使用該 tag，不要沿用本文件的 v3 設定。
