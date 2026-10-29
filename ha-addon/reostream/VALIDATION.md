# Validation record

Date: 2026-10-29. Local host: Linux aarch64. Application source:
`ab698dba24f218be26192d25ff1cc1fd38f59778`.

| Check | Result |
| --- | --- |
| `python3 scripts/check_ha_addon.py` | Passed: exact compiled-source match and Frigate example YAML |
| YAML parsing, shell syntax, Python compile, `git diff --check` | Passed |
| `go vet ./...` with Go 1.26.0 | Passed |
| `go test ./... -timeout 20m` | Passed: all packages |
| Static `CGO_ENABLED=0` Linux builds | Passed for amd64 and arm64; ELF architectures verified |
| Native ARM64 process smoke test | Passed: HTTP status, RTSP OPTIONS, claim POST, 0600 config, config survives restart, management redirects to login, SIGTERM exits 0 |
| Standalone application/streaming source and Dockerfile/compose | Unchanged |
| Docker image builds and container volume recreation smoke | Not run locally: no Docker/Podman engine; configured in CI on native amd64 and aarch64 runners |
| Add-on metadata schema | Passed locally against frenck/action-addon-linter main config.schema.json (Draft 7, format checks); full lint action configured in CI |
| Race tests and standalone Docker build | Configured in CI |
| Real HAOS, Supervisor installation/watchdog/backup/restore | Unverified, hardware required |
| Real cameras, FTP, MQTT broker, Frigate playback/events | Unverified, hardware/services required |

The local Go suite initially failed dependency/socket access inside the
sandbox; it was restarted with network and local sockets allowed. No claim
of successful HAOS or Frigate end-to-end testing is made. Container smoke
checks use an empty camera fleet, not synthetic claims of real camera media.

Publishing was blocked: the GitHub connector returned HTTP 403 Resource not
accessible by integration when creating a blob; the local gh token is invalid.
No remote branch or pull request was created.

Before merging, publish the branch and require every Home Assistant add-on
workflow job to pass.
Before advertising hardware support as tested, complete the acceptance steps
in DOCS.md on HAOS and both CPU architectures. Installation from the default
repository URL becomes available after merging; the branch URL works after
the feature branch is published.
