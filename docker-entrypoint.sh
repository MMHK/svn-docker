#!/bin/sh
# =============================================================================
# svn / svn-admin 共用入口
#
# 目的：讓兩個容器的進程 uid/gid 與 host 上的 /var/svnlocal owner 對齊
# （遠端實測為 1000:1000 / mixmedia），避免用 777 放寬權限。
#
# - PUID / PGID（預設 1000:1000）調整映像內執行帳號的 uid/gid
# - 不會 chown /var/svnlocal 或 /data：host 目錄的 owner 由部署端負責
# - svn：以 root 啟動 Apache（綁 80），worker 落到調整後的帳號
# - svn-admin：調整後用 setpriv 降到非 root 執行
# =============================================================================
set -eu

PUID="${PUID:-${SVN_ADMIN_UID:-1000}}"
PGID="${PGID:-${SVN_ADMIN_GID:-1000}}"
RUN_USER="${SVN_ENTRY_USER:-www-data}"

# Apache 的 ${SVN_REMOTEIP_TRUSTED_PROXY} 必須有值才不會解析失敗；
# 預設是本專案 docker 網路所在的兩個私有網段。不可放寬（見 readme「內網限制」）。
SVN_REMOTEIP_TRUSTED_PROXY="${SVN_REMOTEIP_TRUSTED_PROXY:-172.16.0.0/12 192.168.0.0/16}"
export SVN_REMOTEIP_TRUSTED_PROXY

umask 002

log() {
	printf '[entrypoint] %s\n' "$*" >&2
}

case "$PUID" in
	'' | *[!0-9]*) log "PUID must be a positive integer, got '$PUID'"; exit 1 ;;
esac
case "$PGID" in
	'' | *[!0-9]*) log "PGID must be a positive integer, got '$PGID'"; exit 1 ;;
esac

if ! id "$RUN_USER" >/dev/null 2>&1; then
	log "runtime account '$RUN_USER' does not exist in this image"
	exit 1
fi

if [ "$(id -u)" -eq 0 ]; then
	RUN_GROUP="$(id -gn "$RUN_USER")"

	if [ "$(id -g "$RUN_USER")" != "$PGID" ]; then
		if getent group "$PGID" >/dev/null 2>&1; then
			# 目標 gid 已被別的群組佔用：直接改用該群組，避免 groupmod 失敗
			RUN_GROUP="$(getent group "$PGID" | cut -d: -f1)"
			usermod -g "$RUN_GROUP" "$RUN_USER"
		else
			groupmod -g "$PGID" "$RUN_GROUP"
		fi
	fi

	if [ "$(id -u "$RUN_USER")" != "$PUID" ]; then
		if getent passwd "$PUID" >/dev/null 2>&1; then
			# 目標 uid 已被別的帳號佔用：允許非唯一 uid，避免 usermod 失敗
			usermod -o -u "$PUID" "$RUN_USER"
		else
			usermod -u "$PUID" "$RUN_USER"
		fi
	fi

	log "runtime account: $RUN_USER uid=$(id -u "$RUN_USER") gid=$(id -g "$RUN_USER") group=$RUN_GROUP"

	case "${1:-}" in
		apache2* | /usr/sbin/apache2*)
			mkdir -p /var/run/apache2 /var/lock/apache2 /var/log/apache2
			chown -R "$RUN_USER:$RUN_GROUP" /var/run/apache2 /var/lock/apache2 /var/log/apache2
			exec "$@"
			;;
		*)
			exec setpriv --reuid "$PUID" --regid "$PGID" --init-groups "$@"
			;;
	esac
fi

# 已經以非 root 啟動（例如 compose 的 user: "1000:1000"）：直接執行
log "starting as existing non-root uid=$(id -u) gid=$(id -g)"
exec "$@"
