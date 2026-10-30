# Home Assistant add-on requirements

## Before installing

- **Installation:** Home Assistant OS with Supervisor and access to the add-on
  store. Home Assistant Container alone cannot install this package; use the
  existing standalone reostream Docker deployment there. This package targets
  HAOS; other Supervisor installations have not been validated.
- **CPU:** a 64-bit amd64 or aarch64 HAOS installation. A 64-bit processor with
  a 32-bit OS is insufficient. No armv7, armhf or i386 package is supplied.
- **Permissions:** a Home Assistant administrator must be able to add a custom
  repository, install/start the add-on and read its setup token in the logs.
  Keep Protection mode enabled; the package does not require privileged mode.
- **Build access:** HAOS must be able to pull the Go and Alpine base images,
  fetch the pinned source from GitHub, download Go modules and install Alpine
  packages over the network. Installation builds locally, rather than pulling
  a prebuilt reostream add-on image. DNS and outbound HTTPS must work.
- **Capacity:** allow RAM and disk space for the Go compiler, module cache,
  image build and application data. No measured minimum RAM or build-space
  requirement is established yet. Do not assume idle runtime memory equals
  installation memory. FTP clips and backups need additional free storage;
  the existing clip receiver refuses new clips below 2 GiB free.
- **Network:** allow the HAOS host to reach the cameras. Default incoming TCP
  ports 8560, 8561 and 8562, plus UDP 8000 and 8001, must be available. Optional
  FTP uses TCP 2121–2131. Host networking is enabled; these ports are not
  remapped by Supervisor. Check for other media add-ons using UDP 8000/8001.
  See DOCS.md for outgoing camera ports and configurable listeners.
- **Cameras:** camera login credentials and a supported Reolink Baichuan
  connection. Wired devices need their Baichuan service enabled, normally
  TCP 9000. Battery discovery requires local UDP broadcast reachability.
  Networks separated by VLANs may require manual wired camera addresses.
- **Optional services:** a reachable MQTT broker and account for MQTT events;
  a reachable Frigate/go2rtc installation for recorder integration. Neither
  service is required to start the add-on or open its web interface.

No minimum Home Assistant Core version is declared: the daemon does not use
Core APIs, and no version-specific minimum has been established through HAOS
testing. Use a maintained HAOS/Supervisor release. The package remains
`experimental` until the hardware acceptance checks have been performed.

## Packaging requirements and implementation

This checklist separates required metadata from optional integrations and
records how the current package meets the Supervisor contract.

| Area | Package implementation / acceptance requirement |
| --- | --- |
| Repository | Root `repository.yaml`; nested `ha-addon/reostream/config.yaml` discovered by Supervisor. Publish/merge before using the default repository URL. |
| Required metadata | `name`, quoted `version`, `slug`, `description` and supported `arch` entries are provided. |
| Build context | Self-contained add-on directory; Dockerfile fetches and verifies an exact application commit. No parent-directory COPY or moving source branch. |
| Architectures | `amd64` and `aarch64`; Go maps the latter to `arm64`. Both native container builds and smoke checks passed CI. |
| Runtime dependencies | Static Go application plus Alpine CA certificates, shell and su-exec. No GPU, USB device, ffmpeg or S6 requirement. |
| Startup and shutdown | Default application startup, automatic boot and Supervisor init, foreground exec and a 30-second shutdown timeout. Existing SIGTERM handling is preserved. |
| Health monitoring | Docker `HEALTHCHECK` on `/api/status`; enable Watchdog in the add-on UI. It does not open a stream or wake a camera. |
| Persistent storage | Private Supervisor `/data`; application owns only `/data/reostream`. No shared host mounts. Config, credentials and FTP clips stay there. |
| Backups | `backup: cold` stops the app during copying. Select this add-on in the backup; verify an actual Supervisor restore before declaring it tested. |
| Configuration | Default empty add-on `options`/`schema`; use the existing web UI. No duplicate camera/MQTT options or startup config rewrites. |
| Permissions | Default Protection mode and Docker AppArmor profile; no privileged flags, extra capabilities, devices, Docker socket or Supervisor/Core API access. A custom AppArmor profile is optional and has not been validated. |
| Management UI | Normal HTTP management listener and `webui` link; existing authentication preserved. Ingress is optional and disabled pending compatibility tests. |
| Network exposure | Host networking justified by broadcast discovery and UDP/FTP behavior. Streaming remains unauthenticated; use a trusted network. |
| Store documentation | README, DOCS, CHANGELOG and PNG icon provided. Extra branding/translations are optional; no add-on options currently need translation. |
| Validation | Metadata schema checks, source/version consistency, Go tests/race checks, full lint, both native image builds and container smoke checks passed CI; AMD64 HAOS install/restart/backup/restore also passed. |
| Release readiness | Branch published and PR open. See VALIDATION.md and the PR for passed checks; real camera/integration tests, physical ARM64 HAOS and full host reboot remain outstanding. |

There is no Python `requirements.txt` needed by the add-on: its application
is Go. Development validation uses PyYAML; the workflow pins its version.
A `build.yaml` and Home Assistant base image are not necessary for this
self-contained Dockerfile and single foreground daemon.

## Official references

Requirements checked against the Home Assistant documentation on 2026-10-29:

- [Installation methods and add-on availability](https://www.home-assistant.io/installation/)
- [Repository layout and discovery](https://contributors.home-assistant.io/apps/guides/repository/)
- [Configuration, architecture, lifecycle, storage and backups](https://contributors.home-assistant.io/apps/reference/configuration/)
- [Protection mode and AppArmor](https://contributors.home-assistant.io/apps/guides/security/)
- [Documentation, branding and release stages](https://contributors.home-assistant.io/apps/guides/presentation/)

See [VALIDATION.md](VALIDATION.md) for performed checks and [DOCS.md](DOCS.md)
for installation, network details, Frigate, MQTT and troubleshooting.
