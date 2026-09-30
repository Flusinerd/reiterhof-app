#!/usr/bin/env bash
# Provisions a fresh Debian 12/13 or Ubuntu 22.04/24.04 VPS for the Stallfunk API.
#
# Run as root, from a checkout/copy of the deploy/ directory:
#
#   sudo REITERHOF_DOMAIN=api.example.org \
#        REITERHOF_DB_PASSWORD="$(openssl rand -hex 24)" \
#        REITERHOF_ADMIN_SSH_PUBKEY="ssh-ed25519 AAAA... me@laptop" \
#        REITERHOF_DEPLOY_SSH_PUBKEY="ssh-ed25519 AAAA... github-actions" \
#        ./provision.sh
#
# The script is idempotent: running it again converges the server to the same
# state and never overwrites /etc/reiterhof/api.env once it exists.
#
# Required environment variables:
#   REITERHOF_DOMAIN             public DNS name served by Caddy (A/AAAA record must point here)
#   REITERHOF_DB_PASSWORD        password of the database role (only [A-Za-z0-9._~-], so it
#                                can be embedded into a URL without escaping)
#   REITERHOF_ADMIN_SSH_PUBKEY   public key of the human admin (full sudo, key login only)
#   REITERHOF_DEPLOY_SSH_PUBKEY  public key used by GitHub Actions (may only restart the API)
# Optional:
#   REITERHOF_ADMIN_USER   login name of the admin        (default: admin)
#   REITERHOF_DEPLOY_USER  login name of the deploy user  (default: deploy)
#   REITERHOF_DB_NAME      database name                  (default: reiterhof)
#   REITERHOF_DB_USER      database role                  (default: reiterhof)
set -euo pipefail

# --- Inputs -----------------------------------------------------------------
: "${REITERHOF_DOMAIN:?set REITERHOF_DOMAIN (e.g. api.example.org)}"
: "${REITERHOF_DB_PASSWORD:?set REITERHOF_DB_PASSWORD (e.g. openssl rand -hex 24)}"
: "${REITERHOF_ADMIN_SSH_PUBKEY:?set REITERHOF_ADMIN_SSH_PUBKEY (password login gets disabled!)}"
: "${REITERHOF_DEPLOY_SSH_PUBKEY:?set REITERHOF_DEPLOY_SSH_PUBKEY}"
ADMIN_USER="${REITERHOF_ADMIN_USER:-admin}"
DEPLOY_USER="${REITERHOF_DEPLOY_USER:-deploy}"
DB_NAME="${REITERHOF_DB_NAME:-reiterhof}"
DB_USER="${REITERHOF_DB_USER:-reiterhof}"
export REITERHOF_DB_PASSWORD DB_NAME DB_USER # read by psql via \getenv / the shell

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf '\n==> %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

# --- Preconditions ------------------------------------------------------------
[[ $EUID -eq 0 ]] || die "run as root"
[[ -r /etc/os-release ]] || die "cannot detect the OS"
# shellcheck disable=SC1091
. /etc/os-release
case "${ID}" in
  debian | ubuntu) ;;
  *) die "unsupported OS '${ID}' (Debian or Ubuntu required)" ;;
esac
[[ "$REITERHOF_DB_PASSWORD" =~ ^[A-Za-z0-9._~-]{16,}$ ]] ||
  die "REITERHOF_DB_PASSWORD must be >= 16 chars of [A-Za-z0-9._~-] (try: openssl rand -hex 24)"
[[ "$REITERHOF_DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || die "REITERHOF_DOMAIN looks invalid"
[[ "$DB_NAME" =~ ^[a-z_][a-z0-9_]*$ && "$DB_USER" =~ ^[a-z_][a-z0-9_]*$ ]] ||
  die "database and role names must match [a-z_][a-z0-9_]*"
for f in Caddyfile postgresql.conf.d/reiterhof.conf api.env.example backup.sh restore-test.sh stallfunk-admin.sh \
  systemd/reiterhof-api.service systemd/reiterhof-backup.service systemd/reiterhof-backup.timer; do
  [[ -f "$SCRIPT_DIR/$f" ]] || die "missing $SCRIPT_DIR/$f (copy the whole deploy/ directory)"
done

export DEBIAN_FRONTEND=noninteractive

# --- Base packages ------------------------------------------------------------
log "Installing base packages"
apt-get update -qq
apt-get install -y -qq \
  ca-certificates curl gnupg apt-transport-https debian-keyring debian-archive-keyring \
  ufw unattended-upgrades sudo rclone openssl postgresql-common >/dev/null

# --- Users ----------------------------------------------------------------------
log "Creating users"
# Service account: no login, no home. Owns the running API and the data.
if ! id -u reiterhof >/dev/null 2>&1; then
  useradd --system --user-group --no-create-home --home-dir /nonexistent \
    --shell /usr/sbin/nologin reiterhof
fi

# install_login_user <name> <pubkey>: login user with key-only SSH access.
install_login_user() {
  local name="$1" key="$2"
  if ! id -u "$name" >/dev/null 2>&1; then
    useradd --create-home --shell /bin/bash "$name"
    passwd --lock "$name" >/dev/null # no password, keys only
  fi
  install -d -m 0700 -o "$name" -g "$name" "/home/$name/.ssh"
  printf '%s\n' "$key" >"/home/$name/.ssh/authorized_keys"
  chown "$name:$name" "/home/$name/.ssh/authorized_keys"
  chmod 0600 "/home/$name/.ssh/authorized_keys"
}
install_login_user "$ADMIN_USER" "$REITERHOF_ADMIN_SSH_PUBKEY"
install_login_user "$DEPLOY_USER" "$REITERHOF_DEPLOY_SSH_PUBKEY"
# The deploy user is in the reiterhof group so it can write into /opt/reiterhof.
usermod -aG reiterhof "$DEPLOY_USER"

# Sudo: admin may do everything (key-only login, no password exists), the deploy
# user may only restart the API. Validate with visudo before activating.
sudoers_tmp="$(mktemp)"
trap 'rm -f "$sudoers_tmp"' EXIT
cat >"$sudoers_tmp" <<EOF
# Managed by deploy/provision.sh
${ADMIN_USER} ALL=(ALL) NOPASSWD:ALL
${DEPLOY_USER} ALL=(root) NOPASSWD: /usr/bin/systemctl restart reiterhof-api.service
EOF
visudo -cf "$sudoers_tmp" >/dev/null || die "generated sudoers file is invalid"
install -m 0440 -o root -g root "$sudoers_tmp" /etc/sudoers.d/90-reiterhof

# --- SSH hardening ---------------------------------------------------------------
log "Hardening SSH (no root login, no password auth)"
# sshd uses the FIRST value it sees per keyword and includes sshd_config.d/*.conf in
# lexical order, so this must sort before e.g. Ubuntu's 50-cloud-init.conf.
grep -qE '^\s*Include\s+/etc/ssh/sshd_config\.d/' /etc/ssh/sshd_config ||
  die "/etc/ssh/sshd_config has no Include for sshd_config.d; refusing to guess"
install -d -m 0755 /etc/ssh/sshd_config.d
cat >/etc/ssh/sshd_config.d/00-reiterhof.conf <<'EOF'
# Managed by deploy/provision.sh
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
AllowUsers ADMIN_USER DEPLOY_USER
MaxAuthTries 3
X11Forwarding no
EOF
sed -i "s/ADMIN_USER/${ADMIN_USER}/; s/DEPLOY_USER/${DEPLOY_USER}/" /etc/ssh/sshd_config.d/00-reiterhof.conf
sshd -t || die "sshd configuration invalid; not reloading"
# The service is called "ssh" on Debian/Ubuntu (older releases: "sshd").
systemctl reload ssh 2>/dev/null || systemctl reload sshd

# --- Firewall ----------------------------------------------------------------------
log "Configuring ufw (22, 80, 443)"
ufw default deny incoming >/dev/null
ufw default allow outgoing >/dev/null
ufw limit 22/tcp >/dev/null # SSH with rate limiting
ufw allow 80/tcp >/dev/null # ACME HTTP challenge + redirect
ufw allow 443/tcp >/dev/null
ufw allow 443/udp >/dev/null # HTTP/3
ufw --force enable >/dev/null

# --- Unattended upgrades -------------------------------------------------------------
log "Enabling unattended security upgrades"
cat >/etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
EOF
# Reboot at 04:30 (after the 03:00 backup) if a kernel update requires it.
cat >/etc/apt/apt.conf.d/52reiterhof-unattended <<'EOF'
Unattended-Upgrade::Automatic-Reboot "true";
Unattended-Upgrade::Automatic-Reboot-Time "04:30";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
EOF

# --- PostgreSQL 16 (PGDG repository, identical on Debian and Ubuntu) -------------------
log "Installing PostgreSQL 16"
if [[ ! -f /etc/apt/sources.list.d/pgdg.sources && ! -f /etc/apt/sources.list.d/pgdg.list ]]; then
  /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y >/dev/null
fi
apt-get install -y -qq postgresql-16 postgresql-client-16 >/dev/null

pg_conf_dir=/etc/postgresql/16/main/conf.d
install -d -m 0755 "$pg_conf_dir"
pg_changed=0
if ! cmp -s "$SCRIPT_DIR/postgresql.conf.d/reiterhof.conf" "$pg_conf_dir/reiterhof.conf"; then
  install -m 0644 -o root -g root "$SCRIPT_DIR/postgresql.conf.d/reiterhof.conf" "$pg_conf_dir/reiterhof.conf"
  pg_changed=1
fi
systemctl enable --now postgresql >/dev/null
if ((pg_changed)); then
  systemctl restart postgresql # shared_buffers etc. need a restart
fi

log "Creating database role and database"
# psql reads the password from the environment (\getenv), so it never shows up in
# the process list. \gexec runs the generated statements only when needed.
runuser -u postgres -- env DB_NAME="$DB_NAME" DB_USER="$DB_USER" \
  REITERHOF_DB_PASSWORD="$REITERHOF_DB_PASSWORD" \
  psql -X -q -v ON_ERROR_STOP=1 <<'SQL'
\getenv dbname DB_NAME
\getenv dbuser DB_USER
\getenv pw REITERHOF_DB_PASSWORD
SELECT format('CREATE ROLE %I LOGIN', :'dbuser')
  WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'dbuser') \gexec
SELECT format('ALTER ROLE %I PASSWORD %L', :'dbuser', :'pw') \gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'dbname', :'dbuser')
  WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'dbname') \gexec
SQL

# --- Caddy (official Cloudsmith repository) -----------------------------------------------
log "Installing Caddy"
if [[ ! -f /etc/apt/sources.list.d/caddy-stable.list ]]; then
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' |
    gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    -o /etc/apt/sources.list.d/caddy-stable.list
  apt-get update -qq
fi
apt-get install -y -qq caddy >/dev/null

# --- Directories --------------------------------------------------------------------------
log "Creating directories"
install -d -m 2750 -o "$DEPLOY_USER" -g reiterhof /opt/reiterhof # setgid: new files keep group reiterhof
install -d -m 0750 -o root -g reiterhof /etc/reiterhof
install -d -m 0750 -o reiterhof -g reiterhof /var/lib/reiterhof
install -d -m 0750 -o reiterhof -g reiterhof /var/lib/reiterhof/uploads
install -d -m 0700 -o reiterhof -g reiterhof /var/lib/reiterhof/rclone # rclone config + cache
install -d -m 0750 -o reiterhof -g reiterhof /var/backups/reiterhof

# --- Application configuration --------------------------------------------------------------
log "Writing configuration"
if [[ ! -f /etc/reiterhof/api.env ]]; then
  example="$(<"$SCRIPT_DIR/api.env.example")"
  printf '%s\n' "${example//CHANGE_ME_DB_PASSWORD/$REITERHOF_DB_PASSWORD}" >/etc/reiterhof/api.env
fi
chown root:reiterhof /etc/reiterhof/api.env
chmod 0640 /etc/reiterhof/api.env

# Domain for the Caddyfile ({$REITERHOF_DOMAIN}) via a systemd drop-in.
printf 'REITERHOF_DOMAIN=%s\n' "$REITERHOF_DOMAIN" >/etc/reiterhof/caddy.env
chmod 0640 /etc/reiterhof/caddy.env
chown root:reiterhof /etc/reiterhof/caddy.env
install -d /etc/systemd/system/caddy.service.d
cat >/etc/systemd/system/caddy.service.d/reiterhof.conf <<'EOF'
[Service]
EnvironmentFile=/etc/reiterhof/caddy.env
EOF

# Backup settings (offsite remote etc.). Created once, edited by the owner.
if [[ ! -f /etc/reiterhof/backup.env ]]; then
  cat >/etc/reiterhof/backup.env <<'EOF'
# rclone destination for offsite copies, e.g. "offsite:reiterhof-backups".
# Configure the remote with: sudo -u reiterhof RCLONE_CONFIG=/var/lib/reiterhof/rclone/rclone.conf rclone config
REITERHOF_RCLONE_REMOTE=
# Optional dead man's switch: URL pinged (HTTP GET) after a successful backup
# (e.g. a healthchecks.io check).
REITERHOF_BACKUP_PING_URL=
EOF
fi
chown root:reiterhof /etc/reiterhof/backup.env
chmod 0640 /etc/reiterhof/backup.env

install -m 0644 -o root -g root "$SCRIPT_DIR/Caddyfile" /etc/caddy/Caddyfile
install -m 0755 -o root -g root "$SCRIPT_DIR/backup.sh" /usr/local/sbin/reiterhof-backup
install -m 0755 -o root -g root "$SCRIPT_DIR/restore-test.sh" /usr/local/sbin/reiterhof-restore-test
# Operator tool wrapper: runs /opt/reiterhof/stallfunk-admin (installed by the deploy) as user reiterhof.
install -m 0755 -o root -g root "$SCRIPT_DIR/stallfunk-admin.sh" /usr/local/bin/stallfunk-admin
for unit in reiterhof-api.service reiterhof-backup.service reiterhof-backup.timer; do
  install -m 0644 -o root -g root "$SCRIPT_DIR/systemd/$unit" "/etc/systemd/system/$unit"
done
systemctl daemon-reload

# --- Services ----------------------------------------------------------------------------------
log "Enabling services"
systemctl enable reiterhof-api.service >/dev/null # starts after the first deploy
systemctl enable --now reiterhof-backup.timer >/dev/null
# Caddy: validate first, then (re)load. Without a running API it answers 502 until deployed.
runuser -u caddy -- env REITERHOF_DOMAIN="$REITERHOF_DOMAIN" \
  caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null
systemctl enable caddy >/dev/null
systemctl restart caddy

log "Done. Next steps:"
cat <<EOF
  1. Test SSH login as '${ADMIN_USER}' and '${DEPLOY_USER}' in a NEW terminal before closing this session.
  2. Review /etc/reiterhof/api.env.
  3. Configure the offsite backup remote (see /etc/reiterhof/backup.env).
  4. Trigger the first deploy from GitHub Actions (workflow "Deploy").
  5. Create the first stable and admin: stallfunk-admin help (see deploy/README.md, "Betrieb").
  See deploy/README.md for the full runbook.
EOF
