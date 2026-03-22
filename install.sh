#!/usr/bin/env bash

APP_NAME="iitj-login"
VERSION="3.0.0"

BASE_DIR="$HOME/.local/share/$APP_NAME"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/$APP_NAME.service"
LOGIN_SCRIPT="$BASE_DIR/login.sh"
CRED_FILE="$BASE_DIR/credentials.enc"
KEY_FILE="$BASE_DIR/key.bin"

POST_URL="https://gateway.iitj.ac.in:1003/"
LOGOUT_URL="https://gateway.iitj.ac.in:1003/logout"

# populated by detect_interface / get_nm_connection / get_gateway
INTERFACE=""
CONN=""
GW=""

print_banner() {
cat << "EOF"
 ==========================================
 ______  ______  ________  _____
/      |/      |/        |/     |
$$$$$$/ $$$$$$/ $$$$$$$$/ $$$$$ |
  $$ |    $$ |     $$ |      $$ |
  $$ |    $$ |     $$ | __   $$ |
  $$ |    $$ |     $$ |/  |  $$ |
 _$$ |_  _$$ |_    $$ |$$ \__$$ |
/ $$   |/ $$   |   $$ |$$    $$/
$$$$$$/ $$$$$$/    $$/  $$$$$$/
EOF
cat <<EOF
==========================================
 IITJ Ethernet Auto Login Installer v$VERSION
==========================================
EOF
}

check_dependencies() {
    for cmd in curl openssl systemctl; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            echo "Missing dependency: $cmd — please install it and re-run."
            exit 1
        fi
    done
}

detect_interface() {
    # prefer common ethernet prefixes over wireless (wl) and loopback (lo)
    INTERFACE=$(ip -o link show up 2>/dev/null \
        | awk -F': ' '{print $2}' \
        | grep -E '^(eth|enp|ens|eno|em)' \
        | head -1)

    if [ -z "$INTERFACE" ]; then
        echo "Could not auto-detect ethernet interface."
        read -rp "Enter interface name (e.g. eth0, enp7s0): " INTERFACE
    fi

    echo "Interface: $INTERFACE"
}

get_gateway() {
    GW=$(ip route show dev "$INTERFACE" 2>/dev/null \
        | awk '/^default/ {print $3}' | head -1)

    # fallback to global default route
    if [ -z "$GW" ]; then
        GW=$(ip route 2>/dev/null \
            | awk '/^default via/ {print $3}' | head -1)
    fi
}

get_nm_connection() {
    command -v nmcli >/dev/null 2>&1 || return
    CONN=$(nmcli -g NAME,DEVICE con show --active 2>/dev/null \
        | awk -F: -v iface="$INTERFACE" '$2 == iface {print $1}' \
        | head -1)
}

fix_mac_randomization() {
    # fedora randomizes ethernet mac by default — fortigate auths by mac so this breaks sessions
    if ! command -v nmcli >/dev/null 2>&1; then
        echo "nmcli not found — skipping MAC fix (not needed on most non-Fedora distros)"
        return
    fi

    if [ -z "$CONN" ]; then
        echo "No active NM connection found for $INTERFACE — skipping MAC fix"
        return
    fi

    nmcli connection modify "$CONN" ethernet.cloned-mac-address permanent 2>/dev/null && \
        echo "MAC randomization disabled for: $CONN" || true
}

check_docker_conflict() {
    # docker's default bridge (172.17.x.x) can shadow the fortigate captive portal ip
    # which fortigate's dns interception returns in the same range
    command -v docker >/dev/null 2>&1 || return

    BRIDGE_IP=$(ip addr show docker0 2>/dev/null \
        | awk '/inet / {print $2}' | cut -d/ -f1)
    [ -z "$BRIDGE_IP" ] && return

    if echo "$BRIDGE_IP" | grep -qE '^172\.(1[6-9]|2[0-9]|3[01])\.'; then
        echo ""
        echo "WARNING: Docker bridge (docker0) is using $BRIDGE_IP"
        echo "This conflicts with FortiGate's captive portal IP range."
        echo "Fix it with (requires sudo):"
        echo ""
        echo "  sudo mkdir -p /etc/docker"
        echo '  sudo tee /etc/docker/daemon.json <<'"'"'EOF'"'"
        echo '  { "default-address-pools": [{ "base": "10.200.0.0/16", "size": 24 }] }'
        echo '  EOF'
        echo "  sudo systemctl restart docker && docker network prune -f"
        echo ""
    fi
}

fix_routing() {
    # the captive portal hostname resolves (via fortigate dns interception) to an ip
    # that may route via wifi or a virtual bridge instead of the ethernet interface.
    # pin that ip to the real gateway so logins always reach fortigate.
    [ -z "$CONN" ] && return
    [ -z "$GW" ] && return

    PORTAL_IP=$(getent hosts gateway.iitj.ac.in 2>/dev/null \
        | awk '{print $1}' | head -1)
    [ -z "$PORTAL_IP" ] && return

    ROUTE_DEV=$(ip route get "$PORTAL_IP" 2>/dev/null \
        | awk '{for(i=1;i<NF;i++) if($i=="dev") print $(i+1)}' | head -1)

    if [ "$ROUTE_DEV" != "$INTERFACE" ]; then
        echo "Portal IP $PORTAL_IP routes via '$ROUTE_DEV' — pinning to $INTERFACE..."
        nmcli connection modify "$CONN" \
            +ipv4.routes "$PORTAL_IP/32 $GW" 2>/dev/null || true
        nmcli connection down "$CONN" >/dev/null 2>&1 || true
        sleep 2
        nmcli connection up "$CONN" >/dev/null 2>&1 || true
        echo "Route pinned: $PORTAL_IP → $GW via $INTERFACE"
    else
        echo "Routing OK — portal routes via $INTERFACE"
    fi
}

encrypt_credentials() {
    mkdir -p "$BASE_DIR"
    chmod 700 "$BASE_DIR"

    read -rp "Enter IITJ LDAP Username: " USERNAME
    read -rs -p "Enter IITJ LDAP Password: " PASSWORD
    echo

    openssl rand -base64 32 > "$KEY_FILE"
    chmod 600 "$KEY_FILE"

    printf '%s:%s' "$USERNAME" "$PASSWORD" | \
        openssl enc -aes-256-cbc -pbkdf2 -salt \
        -pass file:"$KEY_FILE" \
        -out "$CRED_FILE"

    chmod 600 "$CRED_FILE"
}

create_login_script() {
    # bake interface into the script at install time
    # runtime variables are escaped with \$ so they expand when login.sh runs
    cat > "$LOGIN_SCRIPT" << EOF
#!/usr/bin/env bash

BASE_DIR="\$HOME/.local/share/iitj-login"
CRED_FILE="\$BASE_DIR/credentials.enc"
KEY_FILE="\$BASE_DIR/key.bin"

POST_URL="$POST_URL"
LOGOUT_URL="$LOGOUT_URL"
INTERFACE="$INTERFACE"

logout() {
    curl --interface "\$INTERFACE" -ks --max-time 5 \\
        "\${LOGOUT_URL}?\$(date +%s)" >/dev/null 2>&1 || true
    exit 0
}

trap logout SIGINT SIGTERM

get_credentials() {
    CREDS=\$(openssl enc -aes-256-cbc -d -pbkdf2 \\
        -in "\$CRED_FILE" \\
        -pass file:"\$KEY_FILE")
    USERNAME=\$(echo "\$CREDS" | cut -d: -f1)
    PASSWORD=\$(echo "\$CREDS" | cut -d: -f2-)
}

# after flushing dns cache, dig queries fortigate's intercepted dns on ethernet
# and gets the internal captive portal ip (172.17.0.3) instead of public ips.
# curl uses systemd-resolved which may cache stale public ips from wifi dns;
# we bypass this by resolving once with dig and passing the ip via --resolve.
get_portal_ip() {
    local ip
    ip=\$(dig +short gateway.iitj.ac.in 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1)
    [ -z "\$ip" ] && ip=\$(ip route show dev "\$INTERFACE" | awk '/^default/ {print \$3}' | head -1)
    echo "\$ip"
}

login_loop() {
    while true; do
        # flush dns cache so fortigate's intercepted dns answer is fresh (not stale wifi-dns public ips)
        resolvectl flush-caches 2>/dev/null || true

        RESP=\$(curl -s --interface "\$INTERFACE" \\
            --connect-timeout 10 --max-time 15 \\
            http://neverssl.com 2>/dev/null || true)

        TOKEN=\$(echo "\$RESP" | grep -o 'fgtauth?[^"]*' | sed 's/fgtauth?//')

        if [ -n "\$TOKEN" ]; then
            echo "[\$(date)] Captive portal detected. Attempting login..."

            # dig runs immediately after flush so fortigate intercepts the query and returns 172.17.0.3
            PORTAL_IP=\$(get_portal_ip)
            echo "[\$(date)] Resolved portal IP: \$PORTAL_IP"

            # --resolve bypasses curl's systemd-resolved cache which may still hold stale public ips
            CURL_RESOLVE="gateway.iitj.ac.in:1003:\${PORTAL_IP}"

            # fetch fgtauth page to extract the actual magic value (learned from iitj-autoproxy)
            FGTAUTH_PAGE=\$(curl -ks --interface "\$INTERFACE" \\
                --resolve "\$CURL_RESOLVE" \\
                --connect-timeout 10 --max-time 15 \\
                "https://gateway.iitj.ac.in:1003/fgtauth?\$TOKEN" 2>/dev/null || true)

            MAGIC=\$(echo "\$FGTAUTH_PAGE" | grep -o 'name="magic" value="[^"]*"' | sed 's/name="magic" value="//;s/"//')
            [ -z "\$MAGIC" ] && MAGIC="\$TOKEN"

            # referer and 4Tredir must point to login?TOKEN — fortigate validates the referer
            REFERER="https://gateway.iitj.ac.in:1003/login?\$TOKEN"

            # post credentials; check response for keepalive? which confirms successful authentication
            POST_RESP=\$(curl --interface "\$INTERFACE" -ks \\
                --resolve "\$CURL_RESOLVE" \\
                --connect-timeout 10 --max-time 15 \\
                -X POST "\$POST_URL" \\
                -H "Content-Type: application/x-www-form-urlencoded" \\
                -H "Referer: \$REFERER" \\
                --data-urlencode "username=\$USERNAME" \\
                --data-urlencode "password=\$PASSWORD" \\
                --data-urlencode "magic=\$MAGIC" \\
                --data-urlencode "4Tredir=\$REFERER" \\
                2>/dev/null || true)

            if echo "\$POST_RESP" | grep -q "keepalive?"; then
                echo "[\$(date)] Login successful."
            else
                echo "[\$(date)] Login POST sent — no keepalive in response (may have failed)."
                echo "[\$(date)] Response: \${POST_RESP:0:300}"
            fi
        else
            echo "[\$(date)] Already authenticated."
        fi

        sleep 300
    done
}

get_credentials
login_loop
EOF

    chmod +x "$LOGIN_SCRIPT"
}

create_service() {
    mkdir -p "$SERVICE_DIR"

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=IITJ LAN Auto Login
After=network-online.target

[Service]
ExecStart=$LOGIN_SCRIPT
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
EOF

    # enable linger so the service runs without an active login session
    loginctl enable-linger 2>/dev/null || true

    systemctl --user daemon-reload
    systemctl --user enable "$APP_NAME"
    systemctl --user start "$APP_NAME"
}

install_app() {
    check_dependencies
    detect_interface
    get_gateway
    get_nm_connection
    fix_mac_randomization
    check_docker_conflict
    fix_routing
    encrypt_credentials
    create_login_script
    create_service
    echo ""
    echo "Installation complete. Service started."
    echo "Run this script again to start / stop / uninstall."
}

start_app() {
    systemctl --user start "$APP_NAME"
    echo "Service started."
}

stop_app() {
    systemctl --user stop "$APP_NAME"
    echo "Service stopped."
}

status_app() {
    systemctl --user status "$APP_NAME" --no-pager
}

uninstall_app() {
    systemctl --user stop "$APP_NAME" 2>/dev/null || true
    systemctl --user disable "$APP_NAME" 2>/dev/null || true
    rm -f "$SERVICE_FILE"
    rm -rf "$BASE_DIR"
    systemctl --user daemon-reload
    echo "Uninstalled."
}

is_installed() {
    systemctl --user list-unit-files 2>/dev/null | grep -q "$APP_NAME.service"
}

is_running() {
    systemctl --user is-active --quiet "$APP_NAME" 2>/dev/null
}

main_menu() {
    print_banner
    echo

    if ! is_installed; then
        echo "1) Install"
        echo "0) Exit"
        read -rp "Choose option: " choice
        case $choice in
            1) install_app ;;
            0) exit 0 ;;
            *) echo "Invalid option." ;;
        esac
    else
        if is_running; then
            echo "1) Stop"
        else
            echo "1) Start"
        fi
        echo "2) Status"
        echo "3) Uninstall"
        echo "0) Exit"
        echo

        read -rp "Choose option: " choice

        case $choice in
            1)
                if is_running; then stop_app; else start_app; fi
                ;;
            2) status_app ;;
            3) uninstall_app ;;
            0) exit 0 ;;
            *) echo "Invalid option." ;;
        esac
    fi
}

main_menu
