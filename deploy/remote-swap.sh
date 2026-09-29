#!/usr/bin/env bash
# Runs ON THE SERVER as the deploy user (piped in by .github/workflows/deploy.yml):
#   ssh deploy@host 'bash -s' < deploy/remote-swap.sh
#
# Expects the new binary at /opt/reiterhof/api.new (uploaded via scp). Swaps it in
# atomically, restarts the service and checks /healthz. If the new version does
# not become healthy, the previous binary is restored and the script fails.
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

sudo systemctl restart reiterhof-api.service
if healthy; then
  echo "Deployment healthy."
  exit 0
fi

echo "ERROR: new version is not healthy." >&2
journalctl -u reiterhof-api.service -n 30 --no-pager >&2 || true
if [[ -f api.prev ]]; then
  echo "Rolling back to the previous binary." >&2
  cp -p api.prev api.rollback.tmp
  mv -f api.rollback.tmp api
  sudo systemctl restart reiterhof-api.service
  healthy && echo "Rollback healthy." >&2 || echo "ERROR: rollback is not healthy either!" >&2
fi
exit 1
