# Changelog

All notable releases for `iitj-login` are tracked here.

## v4.0.0 - 2026-03-23

- Rewrote the tool in Go as a cross-platform single binary.
- Added native installers and service setup for Linux, macOS, and Windows.
- Moved credential storage to AES-256-GCM in the Go implementation.
- Kept the FortiGate login loop, DNS bypass, routing fixes, and Docker conflict checks from the bash line.
- Added release automation so tagged versions publish downloadable binaries.

## v3.1.0 - 2026-03-22

- Added the `/etc/hosts` fix for `gateway.iitj.ac.in` to avoid browser captive-portal failures caused by DNS races.

## v3.0.0 - 2026-03-22

- Promoted the Linux bash implementation to a production-grade installer.
- Added multi-distro support, better DNS handling, and improved fgtauth extraction.
- Added the bootstrap install path for the script-based release line.

## v2.0.0 - 2026-03-22

- Stabilized the bash implementation around the `fgtauth` login flow.
- Added the MAC randomization fix needed for reliable FortiGate re-authentication.

## v1.0.0 - 2026-03-03

- Initial public Linux release of the IITJ LAN auto-login tool.
- Shipped the original bash-based installer and service flow.
