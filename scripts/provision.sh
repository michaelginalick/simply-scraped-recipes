#!/bin/bash
set -euo pipefail

HOST="${1:-107.23.89.139}"

echo "==> Provisioning $HOST"

echo "==> Creating user and directories"
ssh ec2-user@"$HOST" "
  sudo useradd --system --no-create-home --shell /sbin/nologin recipe-app 2>/dev/null || true
  sudo mkdir -p /opt/recipe-app /var/lib/recipe-app
  sudo chown -R recipe-app:recipe-app /opt/recipe-app /var/lib/recipe-app
"

echo "==> Writing recipe-app systemd service"
ssh ec2-user@"$HOST" "sudo tee /etc/systemd/system/recipe-app.service > /dev/null << 'EOF'
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
EOF"

echo "==> Installing Caddy"
ssh ec2-user@"$HOST" "
  curl -fL https://github.com/caddyserver/caddy/releases/download/v2.9.1/caddy_2.9.1_linux_amd64.tar.gz -o /tmp/caddy.tar.gz
  sudo tar -xzf /tmp/caddy.tar.gz -C /usr/local/bin caddy
  sudo chmod +x /usr/local/bin/caddy
"

echo "==> Writing Caddy systemd service"
ssh ec2-user@"$HOST" "sudo tee /etc/systemd/system/caddy.service > /dev/null << 'EOF'
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
EOF"

echo "==> Writing Caddyfile"
ssh ec2-user@"$HOST" "
  sudo mkdir -p /etc/caddy
  sudo tee /etc/caddy/Caddyfile > /dev/null << 'EOF'
simplyscrapedrecipes.com, www.simplyscrapedrecipes.com {
    reverse_proxy localhost:8080
}
EOF"

echo "==> Starting Caddy"
ssh ec2-user@"$HOST" "
  sudo systemctl daemon-reload
  sudo systemctl enable --now caddy
  sudo systemctl status caddy --no-pager
"

echo "==> Deploying binary"
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o /tmp/recipe-api ./cmd/recipe-api
scp /tmp/recipe-api ec2-user@"$HOST":/tmp/recipe-api
ssh ec2-user@"$HOST" "sudo mv /tmp/recipe-api /opt/recipe-app/recipe-api && sudo chmod +x /opt/recipe-app/recipe-api"

echo "==> Starting recipe-app"
ssh ec2-user@"$HOST" "
  sudo systemctl enable --now recipe-app
  sudo systemctl status recipe-app --no-pager
"

echo "==> Attaching static IP"
aws lightsail attach-static-ip \
  --static-ip-name recipe-app-production-ip \
  --instance-name recipe-app-production \
  --profile TERRAFORM_USER || echo "Static IP already attached or attachment failed — check manually"

echo "==> Done. Site should be reachable once DNS propagates to the static IP."
