# IITJ LAN Auto Login

Automatic background login for IIT Jodhpur hostel LAN (FortiGate captive portal).

Keeps your Ethernet session alive by re-authenticating before the ~2h 46m timeout expires.

No more dropped SSH sessions. No more failed downloads. No more broken builds.

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/bootstrap.sh | bash
./install.sh
```

Enter your IITJ LDAP credentials once. The installer handles everything else.

---

## What the installer does

- Auto-detects your ethernet interface
- Disables MAC randomization (critical on Fedora — FortiGate authenticates by MAC)
- Detects and warns about Docker subnet conflicts
- Pins captive portal routing to ethernet (prevents WiFi from stealing packets)
- Adds `gateway.iitj.ac.in` to `/etc/hosts` so the browser portal loads correctly
- Encrypts credentials with AES-256
- Installs and starts a systemd user service
- Enables linger so it runs without an active login session

---

## Managing the service

Re-run the installer for a menu:

```bash
./install.sh
```

Options: Start / Stop / Status / Uninstall

Or use systemctl directly:

```bash
systemctl --user status iitj-login
systemctl --user stop iitj-login
systemctl --user start iitj-login
```

---

## How it works

FortiGate intercepts any plain HTTP request from unauthenticated devices and returns a redirect containing a one-time `fgtauth` token. That token is the `magic` field the login POST requires.

The login loop:

1. Curl `http://neverssl.com` via the ethernet interface
2. If FortiGate intercepts it, extract the `fgtauth` token
3. POST credentials + token to `https://gateway.iitj.ac.in:1003/`
4. Sleep 7200 seconds, repeat

FortiGate stores one session per MAC address. Re-login resets the expiry timer — not a new session.

If neverssl returns the real page (no intercept), the device is already authenticated — skip and sleep.

---

## Known issues and fixes

### Fedora: MAC randomization

Fedora randomizes ethernet MACs by default. FortiGate authenticates by MAC, so every reconnect looks like a new unknown device.

The installer fixes this automatically via:
```bash
nmcli connection modify "<connection>" ethernet.cloned-mac-address permanent
```

### Docker: subnet conflict

Docker's default bridge (`172.17.0.0/16`) can overlap with the IP that FortiGate returns for `gateway.iitj.ac.in` via DNS interception. Traffic destined for the captive portal gets routed into Docker locally instead of reaching FortiGate.

The installer detects this and prints the fix. To apply manually:

```bash
sudo mkdir -p /etc/docker
echo '{"default-address-pools":[{"base":"10.200.0.0/16","size":24}]}' \
  | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
docker network prune -f
```

### WiFi + Ethernet: routing conflict

When both WiFi and Ethernet are active, the captive portal IP can route via WiFi instead of Ethernet. The installer pins the portal IP to the ethernet gateway via a static nmcli route.

### glibc DNS vs kernel DNS: browser portal not loading

`dig` and kernel routing see FortiGate's intercepted DNS (`172.17.0.3`) because those DNS packets go via the ethernet interface. But browsers and GNOME use **glibc** (`getent`) for resolution, which may race WiFi's DNS server and cache the real public IPs for `gateway.iitj.ac.in` instead. Port 1003 doesn't exist on those public IPs — so the captive portal page never loads in the browser, and the GNOME portal popup just spins forever.

Fix: pin the hostname in `/etc/hosts`, bypassing DNS entirely for all processes:

```bash
echo "172.17.0.3 gateway.iitj.ac.in" | sudo tee -a /etc/hosts
```

The installer does this automatically.

### The login script's DNS race condition

The same glibc vs kernel DNS split affects `curl` inside the login script. `curl` calls `getaddrinfo()` (glibc) to resolve hostnames, so it can get public IPs even when `dig` returns `172.17.0.3`. The script now:

1. Calls `resolvectl flush-caches` to clear stale entries
2. Uses `dig +short gateway.iitj.ac.in` immediately after (whose UDP packet FortiGate intercepts on ethernet, returning `172.17.0.3`)
3. Passes that IP to `curl` via `--resolve gateway.iitj.ac.in:1003:<IP>`, bypassing `getaddrinfo` entirely

---

## File locations

```
~/.local/share/iitj-login/   credentials.enc, key.bin, login.sh
~/.config/systemd/user/      iitj-login.service
```

---

## Security

- Credentials encrypted with AES-256-CBC + PBKDF2
- Key stored locally with `600` permissions
- No plaintext credentials anywhere
- No telemetry, no external servers
- Fully open-source — review before running

---

## Requirements

- `curl`, `openssl`, `systemctl`, `dig` (all standard on modern Linux)
- `nmcli` (optional — needed for MAC fix and routing fix, available on NetworkManager distros)

Tested on: Ubuntu 22.04+, Fedora 39+

---

## Version

v3.1.0

---

## License

[MIT](LICENSE)

---

## Disclaimer

Not affiliated with IIT Jodhpur. Use responsibly.
