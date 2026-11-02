#!/bin/sh
set -eu
umask 077
ulimit -c 0
export HOME=/data/home XDG_CONFIG_HOME=/data/home/.config
# fnOS can start Docker resources before the install callback finishes password setup.
# Wait without writing directories or changing ownership until package initialization completes.
if [ -z "${SONGLIB_ADMIN_PASSWORD:-}" ]; then
 ready=0
 for attempt in $(seq 1 60); do
  if [ -f /data/keys/initial-password ] || [ -f /data/keys/management-password.hash ]; then ready=1; break; fi
  sleep 1
 done
 [ "$ready" = 1 ] || { printf '%s\n' '等待飞牛安装初始化超时，请检查应用数据目录权限与安装日志。' >&2; exit 1; }
fi
mkdir -p "$XDG_CONFIG_HOME" /data/lockdown /data/PlumeImpactor/pairing_files
[ -e "$XDG_CONFIG_HOME/PlumeImpactor" ] || ln -s /data/PlumeImpactor "$XDG_CONFIG_HOME/PlumeImpactor"
if [ -d /keep/lib ] && [ ! -e /data/PlumeImpactor/lib ]; then cp -r /keep/lib /data/PlumeImpactor/lib; fi
if [ -d /keep/DeveloperDiskImages ] && [ ! -e /data/DeveloperDiskImages ]; then cp -r /keep/DeveloperDiskImages /data/DeveloperDiskImages; fi
if [ ! -f /data/config.yaml ]; then cp /opt/config.yaml /data/config.yaml; fi
exec /usr/bin/atvloadly server -p 5533 -c /data/config.yaml
