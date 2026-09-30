#!/usr/bin/env bash
# Runs ON THE SERVER as the deploy user (piped in by .github/workflows/deploy.yml):
#   ssh deploy@host 'bash -s' < deploy/remote-swap.sh
#
# Expects the new binary at /opt/reiterhof/api.new (uploaded via scp). Swaps it in
# atomically, restarts the service and checks /healthz. If the new version does
# not become healthy, the previous binary is restored and the script fails.
#
# The operator tool /opt/reiterhof/stallfunk-admin.new (optional) is installed the same
# way (atomic rename, no sudo needed: /opt/reiterhof belongs to the deploy user), but only
# after the API is healthy, so that the tool always matches the running API version.
set -euo pipefail

APP_DIR=/opt/reiterhof
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/healthz}"

cd "$APP_DIR"
[[ -f api.new ]] || { echo "ERROR: $APP_DIR/api.new missing" >&2; exit 1; }
# The service runs as user reiterhof and needs group access (deploy is a member of it).
chgrp reiterhof api.new
chmod 0750 api.new

# Keep the running binary for rollback. cp + mv: the rename is atomic because both
# names are on the same file system.
if [[ -f api ]]; then
  cp -p api api.prev.tmp
  mv -f api.prev.tmp api.prev
fi
mv -f api.new api

healthy() {
  curl -fsS --max-time 3 --retry 15 --retry-delay 2 --retry-connrefused --retry-all-errors \
    "$HEALTH_URL" >/dev/null
}

# install_admin: swap in stallfunk-admin.new (uploaded by the workflow) if present.
# It is executed by user reiterhof via the wrapper /usr/local/bin/stallfunk-admin, hence
# group reiterhof and no write access for the group. mv within one directory is atomic,
# so a running invocation keeps its old inode.
install_admin() {
  [[ -f stallfunk-admin.new ]] || return 0
  chgrp reiterhof stallfunk-admin.new
  chmod 0750 stallfunk-admin.new
  mv -f stallfunk-admin.new stallfunk-admin
  echo "Installed $APP_DIR/stallfunk-admin."
}

sudo systemctl restart reiterhof-api.service
if healthy; then
  echo "Deployment healthy."
  install_admin
  exit 0
fi

echo "ERROR: new version is not healthy." >&2
rm -f stallfunk-admin.new # keep the operator tool matching the API that runs again
journalctl -u reiterhof-api.service -n 30 --no-pager >&2 || true
if [[ -f api.prev ]]; then
  echo "Rolling back to the previous binary." >&2
  cp -p api.prev api.rollback.tmp
  mv -f api.rollback.tmp api
  sudo systemctl restart reiterhof-api.service
  healthy && echo "Rollback healthy." >&2 || echo "ERROR: rollback is not healthy either!" >&2
fi
exit 1
