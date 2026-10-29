# Install and operate

First check the [Home Assistant add-on requirements](REQUIREMENTS.md), including
supported installations, CPU architecture, build access and network ports.

Once this change is merged into the repository's default branch:

1. Open Home Assistant Settings > Add-ons > Add-on Store (called Apps in newer
   versions), then the menu > Repositories.
2. Add `https://github.com/5310h/reostream` and refresh the store.
3. Install **reostream**. Supervisor compiles pinned source on your machine;
   no GHCR image or GitHub login is required. Do not interrupt the first build.
4. Start it; leave **Start on boot** enabled and enable **Watchdog** for health
   recovery. Supervisor also monitors unexpected process exits.
5. Read the first-run claim token in the add-on Log tab. Open Web UI or
   `http://<HAOS-LAN-IP>:8562`, enter the token and save the generated password.
6. Use the existing setup/discovery page to find cameras, then enter their
   credentials. Wired cameras need Baichuan TCP 9000 enabled; battery cameras
   use UDP discovery. Discovery stays within the local broadcast domain;
   across VLANs add wired cameras by IP and allow the required traffic.
7. Stop or restart from the add-on Info page. Allow up to 30 seconds for
   shutdown, so existing camera sessions can be released.

For branch testing after pushing, add
`https://github.com/5310h/reostream#feature/haos-addon` instead. Branch and
default-branch repositories get different Supervisor identities/storage; use
backup/restore deliberately rather than assuming data migrates between them.

# Network and security

Host networking is required for the supported discovery/UDP/FTP setup. There
are no Docker port mappings; clients use the HAOS machine's LAN IP. It also
means the app can reach host services and its listeners are exposed directly
on the host. Only use it on a trusted LAN; restrict access at your router or
network firewall. No privileged mode, devices, extra capabilities, Supervisor
API, Home Assistant API, or shared host directory mounts are requested.

| Traffic | Default | Purpose |
| --- | --- | --- |
| Incoming TCP | 8560 | HTTP MPEG-TS `/driveway.ts`, `/driveway_sub.ts`, frames and `/api/status` |
| Incoming TCP | 8561 | RTSP |
| Incoming TCP | 8562 | Password-protected management UI |
| Incoming UDP | 8000, 8001 | RTSP RTP/RTCP, bound even with TCP clients |
| Incoming TCP, optional | 2121; 2122–2131 | FTP control and passive data, only when enabled |
| Outgoing TCP | Camera 9000, configurable | Wired Baichuan and discovery probes |
| Outgoing UDP / replies | Camera 2015, 2018, negotiated ports | Broadcast search and battery sessions; local ephemeral ports |
| Outgoing TCP | Broker 1883, configurable | MQTT; no broker listener in reostream |
| Outgoing HTTP | Configured Frigate API | Existing detection event create/end calls |

Frigate/go2rtc normally use 8554 for RTSP, avoiding reostream's 8561, but UDP
8000/8001 can conflict with other media services. Check every listener before
installing. The app's UI/config can change TCP listener/FTP addresses; UDP
8000/8001 are fixed in this source version. Resolve their conflict in the
other service. If changing 8560 or 8562, the image health check/Open Web UI
link no longer follows: update the image health check and open the new URL manually,
or maintain matching configuration in a local fork. Keep defaults where possible.

Streaming HTTP and RTSP are **unauthenticated**, as in standalone reostream.
The management password does not protect stream URLs. Do not forward these
ports to the internet. Management is plain HTTP: use a trusted network or a
separately secured reverse proxy. Existing claim-token logging is retained;
restrict log access and never paste config or tokens into public diagnostics.

Ingress and an embedded sidebar panel are disabled pending path-prefix,
WebSocket, cookie and streaming verification. A Home Assistant webpage link
to the normal management URL is optional.

# Storage, backups and updates

The daemon runs as uid/gid 65532. Startup creates `/data/reostream` with mode
0700 and drops root before running Go. The existing config writer creates
`/data/reostream/config.toml` and backups with mode 0600. All camera logins,
management password, MQTT settings and stream definitions remain in this file;
configure them in the original UI, not the add-on options form.

FTP clips default to `/data/reostream/clips`. Keep that default (or another
path below `/data/reostream`) for persistence and ownership. Camera SD-card
recordings stay on the camera. Logs and unclaimed tokens remain in memory.
No configuration is generated or overwritten at startup.

Private `/data` survives restarts, upgrades and container recreation and is
included when this add-on is selected in a Home Assistant backup. The package
uses `backup: cold`: Supervisor stops it during backup so configuration and
clips are copied while the daemon is stopped. Streaming and FTP reception are
unavailable during that backup window. Include the
add-on in backups, secure backups because they contain credentials, and test
restore. FTP clips increase backup size and HAOS disk use; set clip retention
in the UI. The existing receiver refuses new clips below 2 GiB free. Uninstall
can remove private storage: make a backup first. No shared media directory is
mounted, so clips need no separate media backup.

Back up before updating, refresh the store, select Update, then check logs
and stream access. Versions explicitly pin the application commit; a moving
upstream release is never silently substituted. Restore the previous full
add-on backup to roll back settings and clips together. Existing standalone
volumes are not automatically imported. If migrating, copy the entire old
data directory into the add-on's application directory while stopped, retain
uid 65532 ownership, and fix any absolute FTP path.

# Frigate and go2rtc

Name a wired camera `driveway` and enable main and sub streams in reostream,
including their RTSP outputs. Names must match Frigate camera names for event
integration. URLs are `rtsp://<HAOS-LAN-IP>:8561/driveway` (main),
`driveway_sub`, `driveway_extern`, `driveway_zoom`, or `driveway_zoom_sub` when
that camera supplies those streams. There are no RTSP username/passwords.
Use TCP transport for reliability. Battery presence-stream behavior and
`-fpsprobesize 0` requirements are covered in the repository's docs/setup.md.

Example for Frigate with its bundled go2rtc; replace `192.0.2.10` with the HAOS
LAN IP. The loopback URL below is inside Frigate, where go2rtc runs. A separate
go2rtc add-on must instead be addressed by its reachable hostname/IP.

```yaml
mqtt:
  enabled: false  # Optional in Frigate; separate from reostream's event MQTT.
go2rtc:
  streams:
    driveway: rtsp://192.0.2.10:8561/driveway
    driveway_sub: rtsp://192.0.2.10:8561/driveway_sub
cameras:
  driveway:
    ffmpeg:
      inputs:
        - path: rtsp://127.0.0.1:8554/driveway
          input_args: preset-rtsp-restream
          roles: [record]
        - path: rtsp://127.0.0.1:8554/driveway_sub
          input_args: preset-rtsp-restream
          roles: [detect]
    detect:
      enabled: true
    record:
      enabled: true
```

This example is derived from the actual RTSP paths and is syntax checked in
CI; it has **not** been tested end-to-end with Frigate or HAOS hardware.
Use Frigate's config validator for your installed version. For standalone
go2rtc, use just the `streams` mapping under its top-level `streams:` key.
From other add-ons use the HAOS LAN IP, not `localhost` unless the client also
runs on the host network.

Settings > Events can set `frigate` to the reachable internal Frigate API
(e.g. `http://<frigate-host>:5000`). The current app calls
`/api/events/<camera>/<label>/create` and `/api/events/<id>/end`; it has no
Frigate API authentication configuration. Do not expose that unauthenticated
API publicly. Keep names aligned, and check event errors in the UI. No
Frigate code or reostream detection payloads are changed by this package.

# MQTT

Use Settings > Events for broker host:port, user, password and topic prefix.
For Mosquitto add-on use `core-mosquitto:1883` and a dedicated allowed broker
user. If internal DNS is unavailable with host networking, use the HAOS LAN
IP and Mosquitto's enabled LAN listener instead. External brokers use their
reachable LAN host/IP and port. Credentials stay in the application's TOML.

Existing MQTT 3.1.1 is plain TCP with username/password, without TLS or automatic
Home Assistant entity discovery. Default topics are
`reostream/<camera>/<motion|person|vehicle|animal>` with retained `ON`/`OFF`,
and `reostream/<camera>/event` JSON. Existing motion subscriptions remain
unchanged. Configure HA MQTT entities/automations against these topics.
Supervisor MQTT service discovery was considered but is not used: manual UI
configuration avoids new API permissions, broker dependencies and overwriting
existing external-broker settings. Do not enter a `mqtts://` URL.

# Troubleshooting and acceptance

* Startup: inspect Log for TOML validation, filesystem permission and bind
  errors. Never dump config to diagnose startup. Claim tokens change when an
  unclaimed add-on restarts. A bind error commonly means another service owns
  a required TCP or UDP port. A build failure may mean GitHub/Go proxy access,
  storage or RAM is insufficient; the exact source SHA must remain available.
* Discovery: check subnet/VLAN routing, camera Baichuan settings and UDP
  broadcast filtering; use a wired camera IP manually to isolate discovery.
* Streams: check `/api/status` on 8560 and RTSP enablement per camera; allow
  TCP 8561 between Frigate and HAOS. Do not health-check a stream endpoint.
* MQTT: verify broker DNS/port, credentials and ACLs; subscribe to
  `reostream/#`. MQTT does not itself create HA devices/entities.
* Restore: stop, restore a backup including this add-on, start, then verify
  camera list, passwords, MQTT settings and a saved clip.

CI runs Go tests, add-on metadata lint, both architecture image builds and
container smoke checks (first-run HTTP, RTSP listener, claim/config save,
recreation persistence, non-root UID and SIGTERM). See VALIDATION.md for the
actual local results. Real HAOS installation/boot/watchdog/backup/restore,
ARM64 execution on physical hardware, camera discovery/media/two-way audio,
FTP uploads, MQTT broker delivery and Frigate playback/events remain required
hardware acceptance tests. No successful end-to-end test is claimed.
