#!/bin/sh
set -eu
umask 077
ulimit -c 0
export HOME=/data/home XDG_CONFIG_HOME=/data/home/.config
mkdir -p "$XDG_CONFIG_HOME" /data/lockdown /data/PlumeImpactor/pairing_files
[ -e "$XDG_CONFIG_HOME/PlumeImpactor" ] || ln -s /data/PlumeImpactor "$XDG_CONFIG_HOME/PlumeImpactor"
if [ -d /keep/lib ] && [ ! -e /data/PlumeImpactor/lib ]; then cp -r /keep/lib /data/PlumeImpactor/lib; fi
if [ -d /keep/DeveloperDiskImages ] && [ ! -e /data/DeveloperDiskImages ]; then cp -r /keep/DeveloperDiskImages /data/DeveloperDiskImages; fi
if [ ! -f /data/config.yaml ]; then cp /opt/config.yaml /data/config.yaml; fi
exec /usr/bin/atvloadly server -p 5533 -c /data/config.yaml
