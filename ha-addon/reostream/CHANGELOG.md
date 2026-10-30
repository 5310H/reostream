# Changelog

## 0.1.1

Supervisor-validated packaging update using the same application source.
Repairs ownership of restored files within the private application directory,
without following symlinks or changing Supervisor-owned files. Adds a container
regression check for saving settings after restore. Provides a versioned
upgrade from 0.1.0 to verify persistent settings.
Full metadata lint, both native architecture image builds, container
persistence smoke tests, Go race tests and standalone Docker builds pass CI.

## 0.1.0

Initial Supervisor package for amd64 and aarch64. Builds unchanged application
source ab698dba24f218be26192d25ff1cc1fd38f59778. Host networking, private
persistent storage, non-root daemon, web management link and Docker HTTP health check.
HAOS hardware acceptance remains pending; see DOCS.md.

Added Home Assistant prerequisites and packaging requirements checklist. Marked
the initial package experimental and selected cold backups for stopped-file
consistency during backup.

Removed obsolete Supervisor watchdog metadata and redundant defaults after
full linter validation. Added Docker HTTP HEALTHCHECK and container health
assertions; smoke checks now use host networking like the installed add-on.

Use an Alpine Go build stage to reduce on-device build storage; the existing
standalone Dockerfile remains unchanged.
