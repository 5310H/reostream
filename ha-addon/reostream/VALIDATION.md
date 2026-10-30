# Validation record

Date: 2026-10-29. Application source:
`ab698dba24f218be26192d25ff1cc1fd38f59778`.

| Check | Result |
| --- | --- |
| Source/version consistency, Frigate example YAML, script syntax and diff checks | Passed |
| Full Home Assistant add-on linter | Passed locally and in CI; obsolete watchdog field replaced by Docker HEALTHCHECK |
| Go tests and vet | Passed locally and in CI |
| Go race suite | Passed in CI on amd64. Local ARM64 attempt is unsupported by ThreadSanitizer's 39-bit VMA limitation |
| Static binary builds | Passed for Linux amd64 and arm64 |
| Add-on Docker images | Both amd64 and aarch64 build in CI from only the isolated Supervisor add-on context |
| Real container smoke checks, both architectures | Passed in CI: HTTP, RTSP OPTIONS, Docker health, claim/config save, uid 65532, config mode 0600, config and clip marker survive container recreation, SIGTERM exit 0 |
| Standalone Docker deployment | Application source, Dockerfile and compose unchanged; original Dockerfile builds locally on ARM64 and in amd64 CI |
| Actual HAOS installation | Passed on generic-x86-64, HAOS 18.2, Supervisor 2026.10.5, Core 2026.10.4, Docker 29.6.2 |
| Actual Supervisor startup and confinement | HTTP management and RTSP OPTIONS passed; boot auto, watchdog enabled, Protection mode on, default AppArmor and host networking |
| Actual Supervisor restart persistence | Passed: generated management password, authenticated configuration and marker preserved |
| Actual Supervisor cold backup and partial restore | Passed on 0.1.1, including edits after restore. A 0.1.0 restore exposed an unwritable config backup; the 0.1.1 ownership repair fixes it. Home Assistant and all other apps excluded |
| Actual LAN camera discovery | Scan completed and found one candidate; credentials/actual camera media not yet tested |
| Actual watchdog failure recovery | Passed: deliberately failed only the test add-on health check, observed Supervisor restart it automatically, restored the normal listener and verified saved settings |
| Actual Supervisor version upgrade | Passed: 0.1.0 to 0.1.1 preserved the generated management password and configuration fingerprint |
| Physical ARM64 HAOS and full host reboot | Not performed; ARM64 Linux container execution is tested |
| Camera video/audio, camera FTP uploads, broker event delivery and Frigate playback/events | Still require actual camera credentials and integration configuration; no successful end-to-end claim |

The first CI run with all jobs passing is
[Home Assistant add-on run 40974956020](https://github.com/5310H/reostream/actions/runs/40974956020)
for implementation commit `a1b084aa80a2d8d671e500f8692089226d2f4249`.
The [pull request acceptance record](https://github.com/5310H/reostream/pull/1)
tracks subsequent checks, including the restore regression and hardware results.
The [ownership-regression CI run](https://github.com/5310H/reostream/actions/runs/40981432154)
passed all jobs for implementation commit
`ec1979c737e64411fb76b4dae5b32f510c5f5e6a`. Require all checks on the latest
implementation before merging.

The HAOS test uses a separate feature-branch repository identity and temporary
test configuration. It does not overwrite existing standalone data or other
add-on settings. The temporary management password stays in a mode-0600 file
on the SSH add-on; no credentials or claim tokens are published here.

Real camera streams, camera clip uploads and external integration behavior
remain distinct from the empty-fleet/container tests. See DOCS.md for their
configuration and acceptance procedures. The package remains experimental.
