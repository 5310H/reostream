# reostream for Home Assistant OS

Runs the existing reostream application under Supervisor, serving Reolink
Baichuan cameras over HTTP and RTSP with its original management interface,
MQTT alerts, Frigate events and optional FTP clip storage.

Supports **amd64** and **aarch64**, builds locally from a pinned source commit,
and keeps application data in Supervisor's private storage. Installation needs
internet access and enough RAM/disk for a Go build. Initial compilation can
take several minutes on a Pi.

Check the [Home Assistant requirements](REQUIREMENTS.md) before installing.
The package is marked experimental until remaining hardware and camera acceptance testing is complete.

Read [installation and configuration](DOCS.md) and the
[packaging investigation](../INVESTIGATION.md). AMD64 HAOS packaging has been tested; camera integrations and physical
ARM64 HAOS acceptance remain pending. See VALIDATION.md for exact results.
