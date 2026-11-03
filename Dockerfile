# syntax=docker/dockerfile:1

# =============================================================================
# 階段 1：前端產物（Tailwind CSS + app.js）
# 由建置期產生，映像內不使用 repo 裡提交的舊檔
# =============================================================================
FROM node:22-alpine AS assets
WORKDIR /build
COPY package.json yarn.lock .yarnrc.yml build-assets.mjs ./
COPY assets ./assets
# app/views 是 Tailwind 掃描的來源（只有 view 模板需要進 stage）
COPY app/views ./app/views
RUN corepack enable \
 && corepack prepare yarn@4.18.0 --activate \
 && yarn install --immutable \
 && yarn build

# =============================================================================
# 階段 2：PHP 生產依賴（--no-dev，不含 phpunit / phpstan）
# =============================================================================
FROM composer:2 AS vendor
WORKDIR /app
COPY app/composer.json app/composer.lock ./
# autoload 的 classmap / files 會掃 app/src（src/Core/Exceptions.php 等），
# --optimize-autoloader 需要這些檔案存在；只進 stage，不進 runtime
COPY app/src ./src
RUN composer install --no-dev --no-scripts --no-interaction --prefer-dist --optimize-autoloader

# =============================================================================
# 階段 3：runtime
# php:8.3-apache（Debian bookworm）：Apache 2.4 / subversion 1.14 /
# iF.SVNAdmin 已針對 PHP 8 做最小修補（見 readme「PHP 8 相容性」）
# =============================================================================
FROM php:8.3-apache

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      libapache2-mod-svn \
      subversion \
      libonig-dev \
      ca-certificates \
      curl \
  && docker-php-ext-install mbstring \
  && rm -rf /var/lib/apt/lists/*

# mod_dav_svn + Basic Auth + AuthzSVNAccessFile
# access_compat：讓 2013 年的 .htaccess（deny from all）在 Apache 2.4 仍有效
# rewrite：front controller 路由需要
RUN a2enmod dav dav_svn authz_core auth_basic authz_user alias headers access_compat rewrite \
  && rm -f /etc/apache2/sites-enabled/*

# iF.SVNAdmin 直接打進映像，不再從 host bind mount
COPY app/ /var/www-tools/svnadmin/
# 用多階段產物覆蓋：生產 vendor（--no-dev）與建置期前端產物
COPY --from=vendor /app/vendor /var/www-tools/svnadmin/vendor
COPY --from=assets /build/app/public/assets/ /var/www-tools/svnadmin/public/assets/
# data/ 內含 config.ini、userroleassignments.ini；compose 會用 host 目錄蓋上去做持久化
COPY data/ /var/www-tools/svnadmin/data/
RUN mkdir -p /var/www-tools/svnadmin/data/backup

COPY conf/svn.conf /etc/apache2/sites-enabled/000-default.conf
COPY conf/php-svnadmin.ini /usr/local/etc/php/conf.d/zz-svnadmin.ini

WORKDIR /var/www
EXPOSE 80
