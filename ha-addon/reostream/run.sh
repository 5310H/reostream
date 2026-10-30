#!/bin/sh
set -eu
umask 077
# Do not change /data ownership: Supervisor owns options.json alongside us.
mkdir -p /data/reostream
# Supervisor restores can leave backup/clip files with a different owner.
# Walk only our private tree, never follow symlinks or cross mounts, and
# change metadata only for entries whose ownership actually differs.
find /data/reostream -xdev \( ! -user 65532 -o ! -group 65532 \) \
    -exec chown -h 65532:65532 {} +
chmod 0700 /data/reostream
if ! su-exec 65532:65532 test -w /data/reostream; then
    echo 'reostream: /data/reostream is not writable by uid 65532' >&2
    exit 1
fi
# PID/signals pass through to the existing graceful shutdown handler.
exec su-exec 65532:65532 /usr/local/bin/reostream -data /data/reostream
