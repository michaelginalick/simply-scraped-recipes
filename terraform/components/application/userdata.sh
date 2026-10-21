#!/bin/bash
set -euo pipefail

# Install Caddy via official RPM repo
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/setup.rpm.sh' | bash
dnf install -y caddy || {
  # Fallback: install binary directly from GitHub releases
  curl -L https://github.com/caddyserver/caddy/releases/latest/download/caddy_linux_amd64.tar.gz \
    | tar -xz -C /usr/local/bin caddy
  chmod +x /usr/local/bin/caddy
  mkdir -p /etc/caddy /var/log/caddy
  cat > /etc/systemd/system/caddy.service <<'CADDY'
[Unit]
Description=Caddy
After=network.target

[Service]
User=root
ExecStart=/usr/local/bin/caddy run --config /etc/caddy/Caddyfile
ExecReload=/usr/local/bin/caddy reload --config /etc/caddy/Caddyfile
TimeoutStopSec=5s
Restart=on-failure

[Install]
WantedBy=multi-user.target
CADDY
}

# Create app user and directories
useradd --system --no-create-home --shell /sbin/nologin recipe-app
mkdir -p /opt/recipe-app /var/lib/recipe-app
chown recipe-app:recipe-app /opt/recipe-app /var/lib/recipe-app

# Systemd service for the API
cat > /etc/systemd/system/recipe-app.service <<'EOF'
[Unit]
Description=Recipe API
After=network.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
User=recipe-app
WorkingDirectory=/var/lib/recipe-app
ExecStart=/opt/recipe-app/recipe-api
Environment=DB_PATH=/var/lib/recipe-app/recipes.db
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Caddyfile — reverse proxy to the Go server
%{ if domain != "" ~}
cat > /etc/caddy/Caddyfile <<EOF
${domain} {
    reverse_proxy localhost:8080
}
EOF
%{ else ~}
cat > /etc/caddy/Caddyfile <<'EOF'
:80 {
    reverse_proxy localhost:8080
}
EOF
%{ endif ~}

systemctl daemon-reload
systemctl enable --now caddy
# recipe-app service enabled but not started — binary deployed via `make deploy`
systemctl enable recipe-app
