# Setup in detail

The README's Quickstart is enough for most installs. This page covers everything around it.

## Ways to run it

| Method | When |
|---|---|
| `docker run` (README Quickstart) | Default. amd64 and arm64 images. |
| `docker compose up` | Same image, using `docker-compose.yml` from this repo. |
| Static binary | No Docker. linux/amd64 and linux/arm64 on every release. |
| systemd | Plain Linux host, with `contrib/reostream.service`. |
| Build from source | Go 1.26, `go build ./cmd/reostream`. |

### Static binary

```
curl -fsSL -o reostream \
  https://github.com/VoltTech21/reostream/releases/latest/download/reostream-linux-amd64
chmod +x reostream
./reostream -data ./reostream-data
```

It prints the same claim token the container does. No runtime dependencies.

### systemd

Copy the binary to `/usr/local/bin`, create a `reostream` user, and install
`contrib/reostream.service`. It runs with `-data /var/lib/reostream`, which systemd
creates. Read the comment on `KillSignal` and `TimeoutStopSec` before changing either: if
the daemon is killed before it sends its stream-stop messages, the cameras refuse new
connections for several minutes.

## Docker volumes

- The container runs as uid 65532. A named volume (as in `docker-compose.yml`) just works.
  A bind mount to a host directory must be writable by that user:
  `chown -R 65532:65532 ./data` before the first run, or the claim fails to save.
- Mount the whole data directory, not just `config.toml`. Saves are written to a temp
  file and renamed, which needs the directory to be writable.

## Claiming

On first start there is no config. The log prints a one-time token
(`XXXX-XXXX-XXXX-XXXX`) and the URL to open. The token is held in memory only, so a
restart before claiming prints a new one.

The claim page fills in a random 16 character password. Keep it unless you have a reason
not to; a password you type must be at least 16 characters. Submitting writes
`/data/config.toml` and the page asks for that password from then on.

## Config file

Cameras can be added from the page or by editing `/data/config.toml`. When editing by
hand, add to the file that is there; do not replace it. A config without a `[control]`
section is treated as unclaimed and sends you back to the claim screen.

```toml
listen = "0.0.0.0:8560"

[control]
listen = "0.0.0.0:8562"
password = "$REOSTREAM_CONTROL_PASSWORD"

[[camera]]
name = "driveway"
address = "192.0.2.50"
username = "admin"
password = "$REOSTREAM_DRIVEWAY_PASSWORD"
streams = ["main", "sub"]
```

- A value starting with `$` is read from that environment variable.
- Unknown keys are an error at startup, so a typo fails at boot instead of serving 404s.
- Saving from the page validates first and reloads only the cameras that changed.

## Control page

The control port (8562) is the only part with a password. The streaming port (8560) is
open, which is what a recorder needs, so you can firewall the two separately.

It shows each stream's state, plays it live, edits the config, tails the log, and has a
page per camera for settings, floodlight, NTP and timezone. Writes are labeled by how
sure we are they work (proven, unverified, known inert, unsafe), and a 200 from the
camera is shown as "accepted", not "confirmed", unless it was read back. See
`docs/control.md`.

`listen` without a password refuses to start unless `allow_no_password = true` is set.

## RTSP

Off by default. For software that only takes an RTSP URL:

```toml
[rtsp]
listen = "0.0.0.0:8554"

[[camera]]
name = "driveway"
streams = ["main", "sub"]
rtsp = ["main"]
```

Serves `rtsp://host:8554/<cam>`, `<cam>_sub` and `<cam>_extern`. TCP interleaved is the
default; UDP is offered but a lost packet is a corrupt frame. Prefer HTTP where you can:
it reconnects with one request, while RTSP sessions can be left half open by a network
blip. Remember to publish port 8554 from the container.
