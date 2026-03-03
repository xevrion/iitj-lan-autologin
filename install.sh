#!/usr/bin/env bash

set -e

APP_NAME="iitj-login"
VERSION="1.0.0"

BASE_DIR="$HOME/.local/share/$APP_NAME"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/$APP_NAME.service"
LOGIN_SCRIPT="$BASE_DIR/login.sh"
CRED_FILE="$BASE_DIR/credentials.enc"
KEY_FILE="$BASE_DIR/key.bin"

LOGIN_URL="https://gateway.iitj.ac.in:1003/login"
POST_URL="https://gateway.iitj.ac.in:1003/"
LOGOUT_URL="https://gateway.iitj.ac.in:1003/logout"

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
    for cmd in curl openssl sed systemctl; do
        if ! command -v $cmd >/dev/null 2>&1; then
            echo "Missing dependency: $cmd"
            echo "Please install it and re-run."
            exit 1
        fi
    done
}

encrypt_credentials() {
    mkdir -p "$BASE_DIR"
    chmod 700 "$BASE_DIR"

    read -p "Enter IITJ LDAP Username: " USERNAME
    read -s -p "Enter IITJ LDAP Password: " PASSWORD
    echo

    openssl rand -base64 32 > "$KEY_FILE"
    chmod 600 "$KEY_FILE"

    echo "$USERNAME:$PASSWORD" | \
        openssl enc -aes-256-cbc -pbkdf2 -salt \
        -pass file:"$KEY_FILE" \
        -out "$CRED_FILE"

    chmod 600 "$CRED_FILE"
}

create_login_script() {
cat > "$LOGIN_SCRIPT" << 'EOF'
#!/usr/bin/env bash

set -e

LOGIN_URL="https://gateway.iitj.ac.in:1003/login"
POST_URL="https://gateway.iitj.ac.in:1003/"
LOGOUT_URL="https://gateway.iitj.ac.in:1003/logout"

BASE_DIR="$HOME/.local/share/iitj-login"
CRED_FILE="$BASE_DIR/credentials.enc"
KEY_FILE="$BASE_DIR/key.bin"

logout() {
    curl -ks "${LOGOUT_URL}?$(date +%s)" >/dev/null || true
    exit 0
}

trap logout SIGINT SIGTERM

get_credentials() {
    CREDS=$(openssl enc -aes-256-cbc -d -pbkdf2 \
        -in "$CRED_FILE" \
        -pass file:"$KEY_FILE")

    USERNAME=$(echo "$CREDS" | cut -d: -f1)
    PASSWORD=$(echo "$CREDS" | cut -d: -f2-)
}

login_loop() {
    while true; do
        PAGE=$(curl -ks "${LOGIN_URL}?$(date +%s)")
        MAGIC=$(echo "$PAGE" | sed -n 's/.*name="magic" value="\([^"]*\)".*/\1/p')

        if [ -n "$MAGIC" ]; then
            curl -ks -X POST "$POST_URL" \
                -H "Content-Type: application/x-www-form-urlencoded" \
                --data "username=$USERNAME&password=$PASSWORD&magic=$MAGIC&4Tredir=${LOGIN_URL}" \
                >/dev/null

            echo "[`date`] Session refreshed."
        else
            echo "[`date`] Failed to retrieve magic token."
        fi

        sleep 7200
    done
}

get_credentials
login_loop
EOF

chmod +x "$LOGIN_SCRIPT"
}

create_service() {
    mkdir -p "$SERVICE_DIR"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=IITJ LAN Auto Login
After=network-online.target

[Service]
ExecStart=$LOGIN_SCRIPT
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
EOF

    systemctl --user daemon-reload
    systemctl --user enable "$APP_NAME"
    systemctl --user start "$APP_NAME"
}

install_app() {
    check_dependencies
    encrypt_credentials
    create_login_script
    create_service
    echo "Installation complete. Service started."
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
    echo "Uninstalled completely."
}

is_installed() {
    systemctl --user list-unit-files | grep -q "$APP_NAME.service"
}

is_running() {
    systemctl --user is-active --quiet "$APP_NAME"
}

main_menu() {
    print_banner
    echo

    if ! is_installed; then
        echo "1) Install"
        echo "0) Exit"
        read -p "Choose option: " choice
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

        read -p "Choose option: " choice

        case $choice in
            1)
                if is_running; then
                    stop_app
                else
                    start_app
                fi
                ;;
            2) status_app ;;
            3) uninstall_app ;;
            0) exit 0 ;;
            *) echo "Invalid option." ;;
        esac
    fi
}

main_menu