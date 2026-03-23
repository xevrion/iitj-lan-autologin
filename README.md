# IITJ LAN Auto Login

`iitj-login` keeps the IIT Jodhpur hostel Ethernet session alive by logging back into the FortiGate captive portal before the session expires.

It is built for the actual failure modes that make this network annoying to use in practice: expiring sessions, Docker subnet conflicts, MAC randomization, WiFi stealing portal traffic, and DNS returning the wrong address for the gateway.

The current release line is a cross-platform Go binary with support for Linux, macOS, and Windows.

## Why this exists

IITJ hostel LAN does not stay authenticated permanently. If you are on Ethernet, the session typically expires after about 10,000 seconds, which is roughly 2 hours and 46 minutes.

That leads to predictable breakage:

- SSH sessions drop
- downloads fail halfway through
- package managers hang
- long builds die silently
- headless machines become painful to use

This tool runs in the background and re-authenticates automatically so the connection stays usable.

## Features

- single self-contained binary
- Linux, macOS, and Windows support
- background service installation
- Linux and macOS man page installation during setup
- encrypted local credential storage with AES-256-GCM
- automatic Ethernet interface detection
- FortiGate login flow with DNS and routing workarounds
- installer checks for Docker subnet conflicts
- installer fixes the captive portal hostname for browsers

## Installation

### Linux and macOS

```bash
curl -fsSL https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/bootstrap.sh | bash
```

The bootstrap script downloads the latest matching release binary. If no release asset exists for your platform, it falls back to building from source when `go` and `git` are available.

### Windows

Run in PowerShell:

```powershell
irm https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/bootstrap.ps1 | iex
```

The PowerShell bootstrap downloads the latest matching release binary. If no release asset exists for your platform, it falls back to building from source when `go` and `git` are available.

### Build from source

Requires Go 1.21 or newer.

```bash
git clone https://github.com/xevrion/iitj-lan-autologin
cd iitj-lan-autologin
go build -o iitj-login .
./iitj-login install
```

On Windows:

```powershell
git clone https://github.com/xevrion/iitj-lan-autologin
cd iitj-lan-autologin
go build -o iitj-login.exe .
.\iitj-login.exe install
```

## Usage

After installation:

```bash
iitj-login install
```

That setup flow:

1. detects the active Ethernet interface
2. applies the network fixes that are needed on the current machine
3. asks once for IITJ LDAP credentials
4. stores credentials locally in encrypted form
5. installs the background service

Available commands:

```text
iitj-login install
iitj-login uninstall
iitj-login login
iitj-login start
iitj-login stop
iitj-login status
iitj-login version
```

On Linux and macOS, setup also installs a man page:

```bash
man iitj-login
```

## What the installer does

The installer is opinionated because the network problems are specific and repeatable.

### 1. Detects the Ethernet interface

The login requests must leave through Ethernet, not WiFi. The tool detects the active wired interface and binds requests to that path.

### 2. Disables MAC randomization where needed

FortiGate authentication is tied to the device MAC address. If the operating system keeps changing the MAC, the session becomes unreliable.

On Linux systems using NetworkManager, the installer attempts to switch the connection to a permanent MAC address.

### 3. Detects Docker subnet conflicts

Docker commonly uses `172.17.0.0/16` for `docker0`. The captive portal network also uses `172.17.x.x`, which can cause the kernel to route portal traffic locally into Docker instead of the real gateway.

The installer detects this and warns about it.

### 4. Adds a hosts entry for the gateway

The captive portal hostname is pinned to:

```text
172.17.0.3 gateway.iitj.ac.in
```

This avoids DNS races where browsers or other tools resolve the public address instead of the internal captive portal address.

### 5. Pins the route to the captive portal

When WiFi and Ethernet are both active, the route to the portal can go out through the wrong interface. The installer adds a static route so portal traffic stays on Ethernet.

### 6. Stores credentials securely

Credentials are encrypted locally using AES-256-GCM and stored in the user data directory. They are not sent anywhere except the IITJ captive portal.

### 7. Installs a background service

Platform-specific service setup:

- Linux: systemd user service
- macOS: launchd agent
- Windows: Task Scheduler task

## How the login flow works

FortiGate intercepts a normal HTTP request from an unauthenticated client and returns a redirect flow that eventually exposes a one-time authentication token.

The login loop works like this:

1. flush DNS cache where possible
2. request `http://neverssl.com`
3. detect captive portal interception
4. fetch the FortiGate auth page
5. extract the `magic` value needed for login
6. POST IITJ LDAP credentials to `https://gateway.iitj.ac.in:1003/`
7. verify the response indicates a live session
8. sleep and repeat

All login traffic is bound to the Ethernet interface and the tool bypasses unreliable name resolution by using the resolved portal IP directly.

## Known issues

### HTTPS bootstrap can fail before portal login

If FortiGate is intercepting HTTPS before you are authenticated, the bootstrap `curl` command may fail with an SSL error.

Options:

1. log in once in a browser, then run the bootstrap again
2. use `curl -k` for the bootstrap only
3. clone the repository and build locally

### Docker can break portal routing

If Docker is using an overlapping `172.17.x.x` bridge network, portal traffic may never reach FortiGate.

One fix is to move Docker onto a different default pool:

```bash
sudo mkdir -p /etc/docker
echo '{"default-address-pools":[{"base":"10.200.0.0/16","size":24}]}' \
  | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
docker network prune -f
```

### WiFi can steal captive portal traffic

If both WiFi and Ethernet are active, route selection can be wrong. The installer tries to pin the captive portal route to the Ethernet gateway.

## File locations

| Platform | Data directory |
| --- | --- |
| Linux | `~/.local/share/iitj-login/` |
| macOS | `~/Library/Application Support/iitj-login/` |
| Windows | `%APPDATA%\iitj-login\` |

Typical files:

- `credentials.enc`
- `key.bin`
- `config.json`

## Platform support

| Platform | Service integration | Status |
| --- | --- | --- |
| Linux with systemd | systemd user service | supported |
| macOS | launchd agent | supported |
| Windows 10 and 11 | Task Scheduler | supported |

Architectures currently released:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

## Security

- credentials are encrypted locally with AES-256-GCM
- the encryption key stays on the machine
- there is no telemetry
- there are no external services involved
- everything is open source and reviewable

This project still handles real credentials on your machine, so treat it like any other local automation tool with secrets access.

## Releases

Release history is tracked in [CHANGELOG.md](CHANGELOG.md).

Release process for maintainers is documented in [RELEASING.md](RELEASING.md).

## License

[MIT](LICENSE)

## Disclaimer

This project is not affiliated with IIT Jodhpur.
