#!/usr/bin/env bash
set -euo pipefail

# HAProxy SSL/TLS Cert Mismatch Demo
# Sets up HAProxy with a mismatched cert/key pair on a remote Ubuntu server.
# HAProxy fails to start because the cert and key don't match.
# The root cause is discoverable through logs in /var/log.
#
# Usage:
#   ./scripts/haproxy-ssl-debug.sh <ssh-host>          # Setup the broken scenario
#   ./scripts/haproxy-ssl-debug.sh <ssh-host> --cleanup # Tear everything down

usage() {
    echo "Usage: $0 <ssh-host> [--cleanup]"
    echo ""
    echo "  ssh-host   SSH destination (e.g., user@192.168.1.100)"
    echo "  --cleanup  Remove HAProxy, nginx, and all configs"
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

echo "[*] Stopping services..."
systemctl stop haproxy 2>/dev/null || true
systemctl stop nginx 2>/dev/null || true

echo "[*] Removing packages..."
apt-get purge -y haproxy nginx nginx-common >/dev/null 2>&1 || true
apt-get autoremove -y >/dev/null 2>&1 || true

echo "[*] Removing leftover files..."
rm -rf /etc/haproxy/certs
rm -f /etc/nginx/sites-enabled/backend
rm -f /etc/nginx/sites-available/backend
rm -f /etc/rsyslog.d/49-haproxy.conf
rm -f /var/log/haproxy.log
rm -f /tmp/key_a.pem /tmp/key_b.pem /tmp/cert.pem

echo "[*] Restarting rsyslog..."
systemctl restart rsyslog 2>/dev/null || true

echo "[+] Cleanup complete."
CLEANUP_EOF
    exit 0
fi

# ---------------------------------------------------------------------------
# Setup mode
# ---------------------------------------------------------------------------
echo "==> Setting up HAProxy SSL mismatch demo on $SSH_HOST..."

ssh "$SSH_HOST" sudo bash -s <<'SETUP_EOF'
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

# -- Install HAProxy and nginx ------------------------------------------------
echo "[*] Installing haproxy and nginx..."
apt-get update -qq
apt-get install -y -qq haproxy nginx openssl rsyslog >/dev/null

# -- Configure rsyslog so HAProxy gets its own readable log file ---------------
echo "[*] Configuring rsyslog for HAProxy logging..."
cat > /etc/rsyslog.d/49-haproxy.conf <<'RSYSLOG'
local0.* /var/log/haproxy.log
local1.* /var/log/haproxy.log
RSYSLOG

systemctl restart rsyslog
# Make sure the log file exists and is world-readable
touch /var/log/haproxy.log
chmod 644 /var/log/haproxy.log

# -- Configure nginx as backend on port 8080 ----------------------------------
echo "[*] Configuring nginx backend on port 8080..."
cat > /etc/nginx/sites-available/backend <<'NGINX'
server {
    listen 8080;
    location / {
        return 200 'Hello from backend\n';
        add_header Content-Type text/plain;
    }
}
NGINX

rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/backend /etc/nginx/sites-enabled/backend

systemctl restart nginx
echo "[+] nginx running on port 8080."

# -- Write working HTTP-only HAProxy config ------------------------------------
echo "[*] Writing working HTTP-only HAProxy configuration..."
cat > /etc/haproxy/haproxy.cfg <<'HAPROXY'
global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

defaults
    mode http
    log global
    option httplog
    timeout connect 5s
    timeout client 30s
    timeout server 30s

frontend http_front
    bind *:80
    default_backend web_back

backend web_back
    server web1 127.0.0.1:8080 check
HAPROXY

# -- Start HAProxy with working config ----------------------------------------
echo "[*] Starting HAProxy (HTTP-only)..."
systemctl restart haproxy
echo "[+] HAProxy running on port 80."

# -- Generate traffic so logs have entries -------------------------------------
echo "[*] Generating traffic to populate logs..."
for i in $(seq 1 8); do
    curl -s -o /dev/null http://localhost/ || true
done
sleep 2
echo "[+] Traffic generated, logs should have entries."

# -- Generate mismatched SSL cert/key pair ------------------------------------
echo "[*] Generating mismatched SSL certificates..."
mkdir -p /etc/haproxy/certs

# Key pair A - we only use its certificate
openssl genrsa -out /tmp/key_a.pem 2048 2>/dev/null
openssl req -new -x509 -key /tmp/key_a.pem -out /tmp/cert.pem -days 365 \
    -subj "/CN=demo.deer.sh" 2>/dev/null

# Key pair B - we only use its private key
openssl genrsa -out /tmp/key_b.pem 2048 2>/dev/null

# Bundle: cert from A + key from B (MISMATCH)
cat /tmp/cert.pem /tmp/key_b.pem > /etc/haproxy/certs/server.pem
chown root:haproxy /etc/haproxy/certs/server.pem
chmod 640 /etc/haproxy/certs/server.pem

# Clean up temp files so the "evidence" isn't obvious
rm -f /tmp/key_a.pem /tmp/key_b.pem /tmp/cert.pem
echo "[+] Mismatched cert bundle written to /etc/haproxy/certs/server.pem"

# -- Overwrite HAProxy config with broken SSL version -------------------------
echo "[*] Overwriting HAProxy config with SSL frontend..."
cat > /etc/haproxy/haproxy.cfg <<'HAPROXY'
global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

defaults
    mode http
    log global
    option httplog
    timeout connect 5s
    timeout client 30s
    timeout server 30s

frontend https_front
    bind *:443 ssl crt /etc/haproxy/certs/server.pem
    default_backend web_back

backend web_back
    server web1 127.0.0.1:8080 check
HAPROXY

# -- Restart HAProxy (will fail due to cert mismatch) -------------------------
echo "[*] Restarting HAProxy with SSL config..."
systemctl restart haproxy 2>&1 || true

# Give journal a moment to flush
sleep 2

# -- Verify the scenario ------------------------------------------------------
echo ""
echo "=== Verification ==="

echo -n "[check] nginx backend (port 8080): "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080 | grep -q 200; then
    echo "OK (200)"
else
    echo "FAIL"
fi

echo -n "[check] HAProxy process: "
if systemctl is-active --quiet haproxy; then
    echo "running (unexpected - should have failed)"
else
    echo "FAILED to start - as expected"
fi

echo -n "[check] HTTPS on port 443: "
HTTPS_RESULT=$(curl -sk -o /dev/null -w "%{http_code}" https://localhost 2>&1 || true)
if [[ "$HTTPS_RESULT" == "000" || "$HTTPS_RESULT" == "" ]]; then
    echo "connection refused - as expected"
else
    echo "got $HTTPS_RESULT (unexpected)"
fi

echo -n "[check] HAProxy log has entries: "
LOG_LINES=$(wc -l < /var/log/haproxy.log 2>/dev/null || echo 0)
if [[ "$LOG_LINES" -gt 0 ]]; then
    echo "YES ($LOG_LINES lines in /var/log/haproxy.log)"
else
    echo "EMPTY (no log entries found)"
fi

echo -n "[check] Error in journal: "
if journalctl -u haproxy --no-pager -n 50 2>/dev/null | grep -qi "ssl\|cert\|pem\|mismatch\|key values"; then
    echo "YES (SSL error in journalctl)"
else
    echo "checking syslog fallback..."
    if grep -i "haproxy" /var/log/syslog 2>/dev/null | grep -qi "ssl\|cert\|pem\|mismatch"; then
        echo "YES (in syslog)"
    else
        echo "no SSL error found in logs yet"
    fi
fi

echo ""
echo "=== Demo ready ==="
echo "The server has a broken HAProxy setup with a mismatched SSL cert/key pair."
echo "nginx backend works on :8080, HAProxy failed to start, nothing on :443."
echo "/var/log/haproxy.log has entries from when HAProxy was working (HTTP-only phase)."
echo "Root cause is in journalctl -u haproxy and /var/log/syslog."
echo ""
echo "Debugging journey:"
echo "  1. systemctl status haproxy         -> failed/inactive"
echo "  2. /var/log/haproxy.log             -> has old HTTP traffic (proves logging works)"
echo "  3. journalctl -u haproxy --no-pager -> SSL cert/key mismatch error on restart"
echo "  4. openssl s_client -connect localhost:443  -> connection refused (confirms nothing listening)"
echo "  5. ls -la /etc/haproxy/certs/       -> check cert file exists and permissions"
echo "  6. Root cause: cert and private key in PEM bundle don't match"
SETUP_EOF
