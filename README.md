# IITJ LAN Auto Login

Automatic background login for IIT Jodhpur hostel LAN (FortiGate captive portal).

Keeps your Ethernet session alive by re-authenticating before the ~2h 46m timeout expires.

No more dropped SSH sessions. No more failed downloads. No more broken builds.

Cross-platform Go binary — Linux, macOS (Intel + Apple Silicon), Windows.

Versioned binaries are published through GitHub Releases.

---

## Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/bootstrap.sh | bash
```

The bootstrap script downloads the latest matching release binary and falls back to a source build only when no release asset exists for your platform.

Or build from source (requires Go 1.21+):

```bash
git clone https://github.com/xevrion/iitj-lan-autologin
cd iitj-lan-autologin
go build -o iitj-login .
./iitj-login install
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/bootstrap.ps1 | iex
```

The PowerShell bootstrap downloads the latest matching release binary and falls back to a source build only when no release asset exists for your platform.

Or build from source:

```powershell
git clone https://github.com/xevrion/iitj-lan-autologin
cd iitj-lan-autologin
go build -o iitj-login.exe .
.\iitj-login.exe install
```

The installer prompts for your IITJ LDAP credentials once — everything else is automatic.

---

## Commands

```
iitj-login install    # setup wizard (run once)
iitj-login status     # show daemon status
iitj-login start      # start the daemon
iitj-login stop       # stop the daemon
iitj-login uninstall  # remove daemon and stored credentials
```

---

## What the installer does

1. **Detects your ethernet interface** automatically
2. **Disables MAC randomization** — FortiGate authenticates by MAC; randomization breaks sessions (Linux/Fedora via nmcli)
3. **Detects Docker subnet conflicts** — Docker's `172.17.0.0/16` bridge shadows the FortiGate portal IP
4. **Adds `/etc/hosts` entry** — `172.17.0.3 gateway.iitj.ac.in` bypasses DNS races so the browser captive portal loads correctly
5. **Pins routing** — adds a static route for `172.17.0.3` via the ethernet gateway so portal traffic can't leak via WiFi
6. **Encrypts credentials** with AES-256-GCM, stored in your user data dir
7. **Installs daemon**:
   - Linux (systemd): `~/.config/systemd/user/iitj-login.service`
   - macOS (launchd): `~/Library/LaunchAgents/ac.iitj.login.plist`
   - Windows: Task Scheduler task at logon

---

## How it works

FortiGate intercepts any plain HTTP request from unauthenticated devices and returns a JS redirect containing a one-time `fgtauth` token.

The login loop (every 5 minutes):

1. Flush DNS cache (`resolvectl flush-caches` / `dscacheutil` / `ipconfig /flushdns`)
2. `GET http://neverssl.com` — if FortiGate intercepts it, extract the `fgtauth?TOKEN`
3. Fetch `https://gateway.iitj.ac.in:1003/fgtauth?TOKEN` to extract the actual `magic` value
4. `POST` credentials + magic to `https://gateway.iitj.ac.in:1003/`
5. Verify `keepalive?` in the response — confirms successful authentication
6. Sleep 300s, repeat

All HTTP requests are bound to the ethernet interface IP and use the resolved portal IP directly to bypass the glibc DNS race condition.

---

## Known issues and fixes

### MAC randomization (Fedora default)

Fedora randomizes ethernet MACs. FortiGate authenticates by MAC, so every reconnect looks like a new unknown device. Fixed automatically via:

```bash
nmcli connection modify "<connection>" ethernet.cloned-mac-address permanent
```

### Docker subnet conflict

Docker's default bridge (`172.17.0.0/16`) overlaps with FortiGate's portal IP (`172.17.0.3`). The kernel routes portal traffic into Docker locally instead of reaching FortiGate.

The installer detects and warns about this. Manual fix:

```bash
sudo mkdir -p /etc/docker
echo '{"default-address-pools":[{"base":"10.200.0.0/16","size":24}]}' \
  | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
docker network prune -f
```

### WiFi + Ethernet routing conflict

When both interfaces are active, the portal IP can route via WiFi. Fixed by pinning `172.17.0.3/32` to the ethernet gateway as a static route.

### Browser captive portal not loading (glibc DNS race)

Browsers use `getaddrinfo()` (glibc), which may race WiFi's DNS and return public IPs for `gateway.iitj.ac.in`. Port 1003 doesn't exist on those IPs.

Fixed by the `/etc/hosts` entry added during install, which bypasses DNS for all processes.

---

## File locations

| Platform | Data dir |
|----------|----------|
| Linux    | `~/.local/share/iitj-login/` |
| macOS    | `~/Library/Application Support/iitj-login/` |
| Windows  | `%APPDATA%\iitj-login\` |

Files: `credentials.enc`, `key.bin`, `config.json`

---

## Security

- Credentials encrypted with AES-256-GCM
- Key stored locally with `600` permissions, never leaves the machine
- No plaintext credentials anywhere
- No telemetry, no external servers
- Fully open-source — review before running

---

## Platform support

| Platform | Service | Tested |
|----------|---------|--------|
| Linux (systemd) | systemd user service | ✓ Fedora 39+, Ubuntu 22.04+ |
| Linux (non-systemd) | — (cron fallback planned) | — |
| macOS (Intel/M-series) | launchd agent | — |
| Windows 10/11 | Task Scheduler | — |

Architectures: amd64, arm64 (Apple Silicon, Raspberry Pi)

---

## Requirements

**Runtime**: none — single statically-linked binary

**Install-time** (optional, for fixes):
- Linux: `nmcli` for MAC/routing fixes, `sudo` for `/etc/hosts`
- macOS: `sudo` for `/etc/hosts`
- Windows: Administrator for hosts file and routing

---

## Releases

Current binary release line: `v4.0.0`

Release history: [CHANGELOG.md](CHANGELOG.md)

Legacy bash installer: `install.sh` (Linux/systemd only, kept for reference)

---

## License

[MIT](LICENSE)

---

## Disclaimer

Not affiliated with IIT Jodhpur. Use responsibly.
