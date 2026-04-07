#!/usr/bin/env bash
set -euo pipefail

# nginx SSL Cert Path Typo Demo
# Sets up nginx with an HTTPS config that references the wrong certificate path.
# The cert lives at /etc/ssl/certs/app-prod.pem but the config says app.pem.
# nginx fails to start because the file doesn't exist at the configured path.
# The root cause is discoverable through logs and a quick ls comparison.
#
# Usage:
#   ./scripts/nginx-cert-typo.sh <ssh-host>           # Setup the broken scenario
#   ./scripts/nginx-cert-typo.sh <ssh-host> --cleanup # Tear everything down

usage() {
    echo "Usage: $0 <ssh-host> [--cleanup]"
    echo ""
    echo "  ssh-host   SSH destination (e.g., user@192.168.1.100)"
    echo "  --cleanup  Remove nginx, certs, and all configs"
    exit 1
}

if [[ $# -lt 1 ]]; then
    usage
fi

SSH_HOST="$1"
CLEANUP="${2:-}"

# ---------------------------------------------------------------------------
# Cleanup mode
# ---------------------------------------------------------------------------
if [[ "$CLEANUP" == "--cleanup" ]]; then
    echo "==> Cleaning up $SSH_HOST..."
    ssh "$SSH_HOST" sudo bash -s <<'CLEANUP_EOF'
set -euo pipefail

echo "[*] Stopping nginx..."
systemctl stop nginx 2>/dev/null || true

echo "[*] Removing packages..."
apt-get purge -y nginx nginx-common >/dev/null 2>&1 || true
apt-get autoremove -y >/dev/null 2>&1 || true

echo "[*] Removing leftover files..."
rm -f /etc/ssl/certs/app-prod.pem
rm -f /etc/ssl/private/app.key
rm -f /etc/nginx/sites-enabled/app
rm -f /etc/nginx/sites-available/app

echo "[+] Cleanup complete."
CLEANUP_EOF
    exit 0
fi

# ---------------------------------------------------------------------------
# Setup mode
# ---------------------------------------------------------------------------
echo "==> Setting up nginx cert path typo demo on $SSH_HOST..."

ssh "$SSH_HOST" sudo bash -s <<'SETUP_EOF'
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

# -- Install nginx and openssl ------------------------------------------------
echo "[*] Installing nginx and openssl..."
apt-get update -qq
apt-get install -y -qq nginx openssl >/dev/null

# -- Generate a self-signed cert (actual file is app-prod.pem) ----------------
echo "[*] Generating self-signed certificate..."
openssl genrsa -out /etc/ssl/private/app.key 2048 2>/dev/null
openssl req -new -x509 -key /etc/ssl/private/app.key \
    -out /etc/ssl/certs/app-prod.pem -days 365 \
    -subj "/CN=demo.deer.sh" 2>/dev/null
chmod 640 /etc/ssl/private/app.key
echo "[+] Cert written to /etc/ssl/certs/app-prod.pem"

# -- Write a working HTTP config and start nginx ------------------------------
echo "[*] Writing working HTTP config..."
cat > /etc/nginx/sites-available/app <<'NGINX'
server {
    listen 80;
    server_name _;
    location / {
        return 200 'Hello from nginx\n';
        add_header Content-Type text/plain;
    }
}
NGINX

rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/app /etc/nginx/sites-enabled/app

systemctl restart nginx
echo "[+] nginx running on port 80."

# -- Generate traffic so logs have entries ------------------------------------
echo "[*] Generating traffic to populate logs..."
for i in $(seq 1 8); do
    curl -s -o /dev/null http://localhost/ || true
done
sleep 2
echo "[+] Traffic generated, access log should have entries."

# -- Overwrite with broken HTTPS config (typo: app.pem instead of app-prod.pem)
echo "[*] Overwriting nginx config with broken HTTPS config (cert path typo)..."
cat > /etc/nginx/sites-available/app <<'NGINX'
server {
    listen 443 ssl;
    server_name _;

    ssl_certificate     /etc/ssl/certs/app.pem;
    ssl_certificate_key /etc/ssl/private/app.key;

    location / {
        return 200 'Hello from nginx\n';
        add_header Content-Type text/plain;
    }
}
NGINX

# -- Restart nginx (will fail - cert file not found) --------------------------
echo "[*] Restarting nginx with broken SSL config..."
systemctl restart nginx 2>&1 || true

# Give journal a moment to flush
sleep 2

# -- Verify the scenario ------------------------------------------------------
echo ""
echo "=== Verification ==="

echo -n "[check] nginx process: "
if systemctl is-active --quiet nginx; then
    echo "running (unexpected - should have failed)"
else
    echo "FAILED to start - as expected"
fi

echo -n "[check] /etc/ssl/certs/app-prod.pem exists: "
if [[ -f /etc/ssl/certs/app-prod.pem ]]; then
    echo "YES (actual cert file)"
else
    echo "NO (unexpected)"
fi

echo -n "[check] /etc/ssl/certs/app.pem exists: "
if [[ -f /etc/ssl/certs/app.pem ]]; then
    echo "YES (unexpected - typo would not reproduce)"
else
    echo "NO - missing file (typo target)"
fi

echo -n "[check] journalctl has cert error: "
if journalctl -u nginx --no-pager -n 50 2>/dev/null | grep -qi "app.pem\|cannot load certificate\|no such file\|BIO_new_file\|SSL_CTX_use"; then
    echo "YES (cert error in journalctl)"
else
    echo "no cert error found in journal yet"
fi

echo -n "[check] nginx error log has entries from working phase: "
LOG_LINES=$(wc -l < /var/log/nginx/access.log 2>/dev/null || echo 0)
if [[ "$LOG_LINES" -gt 0 ]]; then
    echo "YES ($LOG_LINES lines in /var/log/nginx/access.log)"
else
    echo "EMPTY (no access log entries found)"
fi

echo ""
echo "=== Demo ready ==="
echo "The server has a broken nginx setup with a cert path typo."
echo "Config references /etc/ssl/certs/app.pem but the file is app-prod.pem."
echo "/var/log/nginx/access.log has entries from when nginx was working (HTTP phase)."
echo ""
echo "Debugging journey (read-only):"
echo "  1. systemctl status nginx                    -> failed/inactive, 'cannot load certificate'"
echo "  2. journalctl -u nginx --no-pager -n 50      -> 'open() ... app.pem failed (2: No such file)'"
echo "  3. cat /etc/nginx/sites-enabled/app          -> ssl_certificate /etc/ssl/certs/app.pem"
echo "  4. ls /etc/ssl/certs/app*.pem                -> only /etc/ssl/certs/app-prod.pem exists"
echo "  5. Root cause: config says app.pem, actual file is app-prod.pem (missing -prod suffix)"
SETUP_EOF
