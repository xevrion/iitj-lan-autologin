# IITJ LAN Auto Login

Automatic background login for IIT Jodhpur hostel LAN (FortiGate captive portal).

This tool keeps your Ethernet session alive by periodically refreshing authentication before the 10,000 second timeout expires.

No more dropped SSH sessions.  
No more failed downloads.  
No more random build interruptions.

---

## What It Does

- Reverse-engineers IITJ FortiGate login flow
- Extracts dynamic `magic` token
- Re-authenticates automatically every ~2 hours
- Runs as a systemd user service
- Stores credentials encrypted (AES-256)
- Auto-starts on login
- Provides Start / Stop / Status / Uninstall options

---

## Installation

Run:

```bash
curl -fsSL https://raw.githubusercontent.com/xevrion/iitj-lan-autologin/main/install.sh | bash
```

You will be prompted once for:

- IITJ LDAP Username
- IITJ LDAP Password

The installer will:

- Encrypt credentials
- Create background login script
- Register systemd user service
- Start the service

---

## Usage

After installation, re-run the installer to access:

- Start
- Stop
- Status
- Uninstall

Example:

```bash
bash install.sh
```

---

## How It Works

1. Requests the FortiGate login page.
2. Extracts the hidden `magic` token.
3. Submits credentials via POST.
4. Sleeps for 7200 seconds.
5. Repeats.

FortiGate stores one active session per device (MAC), so re-login simply resets the expiry timer.

---

## File Locations

Credentials and scripts are stored locally:

```
~/.local/share/iitj-login/
```

Systemd user service:

```
~/.config/systemd/user/iitj-login.service
```

---

## Security

- Credentials are encrypted using AES-256.
- Encryption key stored locally with restricted permissions.
- No credentials are stored in plaintext.
- No telemetry.
- No external servers.
- Fully open-source and auditable.

Before running `curl | bash`, always review the source code.

---

## Uninstall

Run:

```bash
bash install.sh
```

Select **Uninstall**.

This removes:

- Background service
- Encrypted credentials
- All related files

---

## Version

Current version: v1.0.0

---

## Disclaimer

This tool is not affiliated with IIT Jodhpur.

Use responsibly and at your own discretion.