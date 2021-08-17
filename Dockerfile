FROM alpine:3.7

MAINTAINER SamZhou<sam@mixmedia.com>

WORKDIR /var/svndata

# Install and configure Apache WebDAV and Subversion
RUN apk --no-cache add apache2 apache2-utils apache2-webdav mod_dav_svn subversion \
    apache2-ssl \
    curl \
    php7-apache2 \
    php7-bcmath \
    php7-bz2 \
    php7-common \
    php7-ctype \
    php7-curl \
    php7-dom \
    php7-iconv \
    php7-mbstring \
    php7-openssl \
    php7-phar \
    php7-session \
    php7-xml \
&& mkdir -p /var/www-tools/svnadmin \
&& mkdir -p /run/apache2
ADD vh-davsvn.conf /etc/apache2/conf.d/
ADD svnadmin /var/www-tools/svnadmin
ADD run.sh /

RUN chmod +x /run.sh
EXPOSE 80

ENV WWW_UID=1000
ENV WWW_GID=1000
ENV SVN_USER=mixmedia
ENV SVN_PWD=123456


# Define default command
CMD ["/run.sh"]
