#!/bin/sh

## add www user
addgroup -g "$WWW_GID" "www"
adduser -D -g "" -h /var/svndata -G "www" -H -u "$WWW_UID" "www"

## replace httpd running user
sed -i -e "s/User apache/User www/g" /etc/apache2/httpd.conf
sed -i -e "s/Group apache/Group www/g" /etc/apache2/httpd.conf

[ ! -d /var/svndata ] && { echo "svndata not found!" ; exit 1; }
[ ! -f /var/svndata/svn-access-file ] && { echo -e "[groups]\nadmin=$SVN_USER" > /var/svndata/svn-access-file; }
[ ! -f /var/svndata/svn-auth-file ] && { touch /var/svndata/svn-auth-file; }
[ -f /var/svndata/svn-auth-file ] && { htpasswd -b /var/svndata/svn-auth-file "$SVN_USER" "$SVN_PWD"; }


httpd -D FOREGROUND
