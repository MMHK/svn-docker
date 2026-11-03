# SVN + iF.SVNAdmin 單一容器映像
# php:8.3-apache（Debian bookworm）：Apache 2.4.62 / subversion 1.14 /
# iF.SVNAdmin 已針對 PHP 8 做最小修補（見 readme「PHP 8 相容性」）
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
RUN a2enmod dav dav_svn authz_core auth_basic authz_user alias headers access_compat \
 && rm -f /etc/apache2/sites-enabled/*

# iF.SVNAdmin 直接打進映像，不再從 host bind mount
COPY app/ /var/www-tools/svnadmin/
# data/ 內含 config.ini、userroleassignments.ini；compose 會用 host 目錄蓋上去做持久化
COPY data/ /var/www-tools/svnadmin/data/
RUN mkdir -p /var/www-tools/svnadmin/data/backup

COPY conf/svn.conf /etc/apache2/sites-enabled/000-default.conf
COPY conf/php-svnadmin.ini /usr/local/etc/php/conf.d/zz-svnadmin.ini

WORKDIR /var/www
EXPOSE 80
