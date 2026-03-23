# Changelog

All notable releases for `iitj-login` are tracked here.

## v4.0.4 - 2026-03-23

- Fixed the Windows PowerShell bootstrap on environments where `RuntimeInformation.OSArchitecture` is unavailable.
- Switched architecture detection to older-compatible Windows and PowerShell signals first, with the newer .NET path only as fallback.

## v4.0.3 - 2026-03-23

- Added terminal-aware color to user-facing CLI output.
- Embedded recent service logs directly into `status` on supported platforms.
- Improved status wording so common states read more naturally for non-technical users.

## v4.0.2 - 2026-03-23

- Replaced raw service-manager dumps in `status` with a readable cross-platform summary.
- Show install state, service state, configured interface details, credential presence, and platform-specific log guidance in one view.
- Make Linux status degrade cleanly when live systemd state is unavailable outside a normal user session.

## v4.0.1 - 2026-03-23

- Added a real `man 1` page and install it during setup on supported systems.
- Ship the man page as a release artifact.
- Remove the installed man page during `uninstall`.
- Fix `uninstall` to also remove stored application data as previously documented.

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
