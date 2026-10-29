# Investigation before implementation

Inspected checkout ab698dba24f218be26192d25ff1cc1fd38f59778 (606 tracked
files): cmd, internal packages, embedded web assets, third_party/gortsplib,
scripts, Dockerfile, compose, release/test workflows and setup documentation.

* The module still uses github.com/VoltTech21/reostream; origin is
  github.com/5310H/reostream. Do not change module imports or streaming logic.
* cmd/reostream is a pure Go 1.26 daemon (CGO disabled in existing Dockerfile).
  Web templates/assets are embedded. No ffmpeg, GPU, devices or external
  runtime service is required. Existing distroless standalone image is untouched.
* main.go defaults: HTTP 8560, RTSP 8561 and control 8562 on all interfaces.
  internal/rtsp/server.go also binds UDP 8000/8001 for RTP/RTCP, even when
  clients use TCP. These fixed UDP ports can conflict with other add-ons.
* Wired Baichuan uses outgoing TCP 9000 (config.NormalizeAddr). Discovery
  scans local subnets (internal/control/discover.go). internal/bcudp searches
  broadcast UDP 2015; connections try UDP 2018/2015 and use ephemeral local
  sockets and camera-negotiated session ports. No multicast listener found.
* internal/control/clips.go and internal/ftpd implement optional FTP:
  TCP 2121 and passive TCP 2122-2131 by default, configurable. It is off
  without [ftp]. Clips default beside config.toml in clips/.
* -data controls config.toml location. Control writes atomic saves and backup
  files beside it, with default mode 0600. Camera credentials, stream settings,
  MQTT and management password live in TOML. Logs and claim token are in memory;
  the existing first-run token is deliberately printed for the operator.
* Recordings on camera SD remain on the camera; downloaded FTP clips are local.
  Persisting the entire application directory preserves configuration and clips.
* internal/mqtt implements MQTT 3.1.1 over plain TCP, broker host:port and
  username/password; no TLS, HA discovery publication or Supervisor discovery.
  internal/control/events.go retains existing topics and Frigate create/end API.
* RTSP paths are camera name (main), name_sub, name_extern, name_zoom and
  name_zoom_sub. HTTP uses /name.ts and /name_sub.ts. Stream servers have no
  authentication. Control has claim-token setup and password authentication.
* SIGTERM is handled already; camera session shutdown needs a grace period.
  /api/status on HTTP 8560 is a cheap health endpoint, without a stream slot.
* Ingress is not enabled: absolute web routes, cookies, streaming and WebSocket
  paths require proxy-prefix/authentication testing. A normal webui link works.

## Packaging decision

Supervisor builds with the add-on directory as context. Use a two-stage build
that fetches the exact original source SHA above, verifies HEAD, and compiles
it; never COPY parent directories or fetch a moving branch. No registry release
is needed for installation. CI checks compiled source paths against this pin,
so subsequent application changes require an explicit pin/version update.
Go cross-compiles amd64/arm64; Supervisor BUILD_ARCH aarch64 maps to Go arm64.
Alpine supplies CA certificates, a shell and su-exec; the Go application runs
as uid 65532 after initializing only /data/reostream. No HA base/S6 is needed
for one foreground process; use Supervisor's default init and exec throughout.
Host networking preserves broadcast discovery, UDP sessions and FTP address
advertisement. No host mounts, API access, devices or privileged mode needed.

References checked 2026-10-29:
https://contributors.home-assistant.io/apps/reference/configuration/
https://contributors.home-assistant.io/apps/guides/repository/
Supervisor discovers nested add-on folders and builds locally when image is
omitted. /data is private persistent add-on storage included in backups.

## Maintainer update procedure

Choose and push an application commit to this repository, then update
REOSTREAM_SOURCE and the OCI revision label in the add-on Dockerfile. Bump
config.yaml version, the io.hass.version label and the ldflags version string
together, and record the source SHA in CHANGELOG.md. Run
`python3 scripts/check_ha_addon.py` (PyYAML required). It compares the compiled
source paths against the selected commit, including the vendored RTSP fork.
Go dependencies are fixed by go.mod/go.sum; toolchain and Alpine release tags
are explicit. OS package revisions/base tag contents are not digest locked;
this is source/version reproducibility, not a claim of bit-identical images.

Local packaging checks with Docker:

```sh
docker build --build-arg BUILD_ARCH=amd64 -t reostream-addon:test ha-addon/reostream
python3 scripts/smoke_ha_addon.py reostream-addon:test
```

Use BUILD_ARCH=aarch64 on an ARM64 machine. CI uses native runners for both
architectures, so no QEMU assumption leaks into Supervisor installation.
The smoke check maps loopback ports in bridge networking to isolate CI; the
metadata selects host networking on HAOS, which needs hardware validation.
