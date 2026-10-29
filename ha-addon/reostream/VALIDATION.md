# Validation record

Date: 2026-10-29. Local host: Linux aarch64. Application source:
`ab698dba24f218be26192d25ff1cc1fd38f59778`.

| Check | Result |
| --- | --- |
| `python3 scripts/check_ha_addon.py` | Passed: exact compiled-source match and Frigate example YAML |
| YAML parsing, shell syntax, Python compile, `git diff --check` | Passed |
| Requirements follow-up: experimental stage and cold backup metadata | Schema and source/version consistency checks passed; actual Supervisor backup remains unverified |
| `go vet ./...` with Go 1.26.0 | Passed |
| `go test ./... -timeout 20m` | Passed: all packages |
| Static `CGO_ENABLED=0` Linux builds | Passed for amd64 and arm64; ELF architectures verified |
| Native ARM64 process smoke test | Passed: HTTP status, RTSP OPTIONS, claim POST, 0600 config, config survives restart, management redirects to login, SIGTERM exits 0 |
| Standalone application/streaming source and Dockerfile/compose | Unchanged |
| Docker image builds and container volume recreation smoke | Temporary Docker daemon provisioned for local ARM64 build; completion pending. Native amd64 and aarch64 runners also configured in CI |
| Add-on metadata schema and full linter | Passed locally using frenck/action-addon-linter v2 lint.py and the published schema; redundant defaults and obsolete watchdog metadata corrected |
| Race tests | Attempted locally; ThreadSanitizer rejects this host (39-bit VMA, requires 48-bit). Native GitHub runner check remains required |
| Standalone Docker build | Configured in CI |
| Real HAOS, Supervisor installation/watchdog/backup/restore | Unverified, hardware required |
| Real cameras, FTP, MQTT broker, Frigate playback/events | Unverified, hardware/services required |

The local Go suite initially failed dependency/socket access inside the
sandbox; it was restarted with network and local sockets allowed. No claim
of successful HAOS or Frigate end-to-end testing is made. Container smoke
checks use an empty camera fleet, not synthetic claims of real camera media.

Publishing initially failed through the GitHub connector (HTTP 403). A later
git push authenticated through VS Code but GitHub rejected the new workflow
because its OAuth token lacks workflow scope. Browser reauthentication has
been requested. No remote branch or pull request exists yet.

Home Assistant at http://192.168.69.7:8123 could not be reached from this
environment; hardware acceptance still needs a reachable HAOS instance.

Before merging, publish the branch and require every Home Assistant add-on
workflow job to pass.
Before advertising hardware support as tested, complete the acceptance steps
in DOCS.md on HAOS and both CPU architectures. Installation from the default
repository URL becomes available after merging; the branch URL works after
the feature branch is published.
