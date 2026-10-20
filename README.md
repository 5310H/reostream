# reostream

Pulls video from Reolink cameras over their native Baichuan protocol and serves it as
MPEG-TS over HTTP. Built for Frigate; works with anything that plays a URL.

```
camera --Baichuan(9000)--> reostream --HTTP MPEG-TS--> Frigate / mpv / VLC
```

## Why

Reolink's RTSP server is broken on some models. On a Duo 3 it gave green blocks, grey
frames and smearing. Same camera, same encoder, same thirty minutes, both transports:

| transport | segments | segments with bitstream errors |
|---|---|---|
| Baichuan | 184 | 0 |
| RTSP | 178 | 147 |

Reolink's own NVR looks fine because it uses Baichuan, not RTSP. A fisheye here also loses
about 21% of its frames over RTSP. Method in `docs/measurements.md`.

Neolink did this first and made the protocol understandable. It has had no commits since
January 2025. reostream is a new implementation, not a fork.

## Quickstart

```
docker run -d --name reostream --restart unless-stopped \
  -v reostream-data:/data \
  -p 8560:8560 -p 8562:8562 \
  ghcr.io/volttech21/reostream:latest

docker logs reostream
```

1. The log prints a one-time token. Open `http://<host>:8562/claim` and paste it.
2. Keep the password it gives you.
3. Add a camera: its IP, `admin`, and its password. Port 9000 on the camera must be
   reachable; it is on by default.

Your stream is at `http://<host>:8560/<camera>.ts`.

Prefer compose, a plain binary or systemd? See `docs/setup.md`.

## Using it with Frigate

```yaml
cameras:
  driveway:
    ffmpeg:
      inputs:
        - path: http://reostream:8560/driveway_sub.ts
          roles: [detect]
        - path: http://reostream:8560/driveway.ts
          roles: [record, audio]
```

No go2rtc entry is needed for detect or record. Keep go2rtc for Frigate's browser live
view if your main stream is HEVC.

## Endpoints

| Endpoint | |
|---|---|
| `/<cam>.ts` | main stream |
| `/<cam>_sub.ts` | sub stream |
| `/<cam>_extern.ts` | extern: the middle stream (about 896x512 H.264) most clients never offer |
| `/<cam>.keyframe` | latest keyframe, raw |
| `/api/status` | per-stream JSON: connected, fps, bitrate, clients, reconnects |
| `/metrics` | the same for Prometheus |

RTSP output is available too, off by default. See `docs/setup.md`.

## What it does

- One connection per stream, held open whether anyone is watching or not.
- Timestamps from the camera's own clock, so frame rates come out right.
- A slow client is disconnected, never buffered. No client can stall a camera.
- New clients start at a keyframe. Audio is carried when the camera sends it.
- Sessions are closed cleanly. A camera left with an open session refuses connections
  for minutes.
- A web page on port 8562 for status, live view, config, logs and camera settings.

No ffmpeg, GStreamer or RTSP server in the serving path.

## Tested on

One fleet of eight wired cameras, four models, one firmware generation: 23 streams,
69 Mbps, about 60% of one CPU core, 29-59 MB of memory. A 7 hour soak over all 23 streams
had zero restarts and flat memory. Numbers in `docs/measurements.md`.

Not yet tested:

- **Battery cameras.** They sleep and wake, and the protocol client has not seen them yet.
  Reports welcome.
- Other people's cameras and firmware.
- Multi-day uptime.

## Camera control

`reocam` reads and changes camera settings over the same protocol. Streaming does not
need it.

```
reocam -address 192.0.2.50 -password secret probe
```

`probe` reports which messages the camera answers. See `docs/control.md` for reading
and writing settings and what to watch out for.

## More

- `docs/setup.md`: other ways to run it, the config file, the control page, RTSP
- `docs/control.md`: camera control with `reocam`
- `docs/protocol.md`: what the Baichuan wire actually does
- `docs/measurements.md`: how every number above was measured

## License

AGPL v3. Free to use, run and modify. If you distribute a modified version or run one as a
service for others, you must publish your changes.

A commercial license without that obligation is available. See `COPYRIGHT`.
