#!/bin/sh
set -eu
umask 077
# Do not change /data ownership: Supervisor owns options.json alongside us.
mkdir -p /data/reostream
chown 65532:65532 /data/reostream
chmod 0700 /data/reostream
if ! su-exec 65532:65532 test -w /data/reostream; then
    echo 'reostream: /data/reostream is not writable by uid 65532' >&2
    exit 1
fi
# PID/signals pass through to the existing graceful shutdown handler.
exec su-exec 65532:65532 /usr/local/bin/reostream -data /data/reostream
