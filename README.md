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

- `curl`, `openssl`, `systemctl` (all standard on modern Linux)
- `nmcli` (optional — needed for MAC fix and routing fix, available on NetworkManager distros)

Tested on: Ubuntu 22.04+, Fedora 39+

---

## Version

v3.0.0

---

## License

[MIT](LICENSE)

---

## Disclaimer

Not affiliated with IIT Jodhpur. Use responsibly.
