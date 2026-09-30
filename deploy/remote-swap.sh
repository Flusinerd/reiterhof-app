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
#
# The web app (PWA) is installed the same way: the optional tarball /opt/reiterhof/web.new.tar.gz
# (Expo web export) is unpacked to /var/www/stallfunk/releases/<RELEASE_SHA> and the symlink
# /var/www/stallfunk/current is switched atomically (ln -sfn + mv -T). Caddy serves "current".
# The last 3 releases are kept, so a rollback is one symlink switch (see deploy/README.md).
# RELEASE_SHA (the deployed commit) is passed in by the workflow:
#   ssh deploy@host "RELEASE_SHA=<sha> bash -s" < deploy/remote-swap.sh
# No sudo is needed: /var/www/stallfunk belongs to the deploy user (provision.sh).
set -euo pipefail

APP_DIR=/opt/reiterhof
WEB_ROOT="${WEB_ROOT:-/var/www/stallfunk}"
WEB_KEEP=3
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

# install_web: unpack web.new.tar.gz (uploaded by the workflow) as a new release and make it
# current. The release directory is built under a temporary name and renamed into place, so a
# release directory is always complete. The symlink is replaced by renaming a new one over it.
install_web() {
  [[ -f web.new.tar.gz ]] || return 0
  [[ "${RELEASE_SHA:-}" =~ ^[0-9a-f]{7,64}$ ]] || { echo "ERROR: RELEASE_SHA missing or invalid" >&2; return 1; }
  local releases="$WEB_ROOT/releases" release incoming
  release="$releases/$RELEASE_SHA"
  incoming="$releases/.incoming-$RELEASE_SHA"
  rm -rf "$releases"/.incoming-* # leftovers of an aborted run (deploys never run in parallel)
  if [[ -d "$release" ]]; then
    echo "Web release $RELEASE_SHA is already installed; reusing it."
  else
    mkdir "$incoming"
    tar -xzf web.new.tar.gz -C "$incoming" --no-same-owner --no-same-permissions
    [[ -f "$incoming/index.html" ]] || { echo "ERROR: web archive has no index.html" >&2; rm -rf "$incoming"; return 1; }
    chmod -R u=rwX,go=rX "$incoming" # Caddy (user caddy) only reads
    mv -T "$incoming" "$release"
  fi
  touch "$release" # newest release = most recently activated; the retention below goes by this
  ln -sfn "$release" "$WEB_ROOT/current.new"
  mv -T "$WEB_ROOT/current.new" "$WEB_ROOT/current"
  rm -f web.new.tar.gz
  echo "Web release $RELEASE_SHA is current."

  # Keep the newest WEB_KEEP releases (the current one is always among them).
  local old
  while IFS= read -r old; do
    [[ "$old" != "$(readlink "$WEB_ROOT/current")" ]] || continue
    rm -rf -- "$old"
  done < <(find "$releases" -mindepth 1 -maxdepth 1 -type d ! -name '.*' -printf '%T@ %p\n' |
    sort -rn | tail -n +$((WEB_KEEP + 1)) | cut -d' ' -f2-)
}

sudo systemctl restart reiterhof-api.service
if healthy; then
  echo "Deployment healthy."
  install_admin
  install_web
  exit 0
fi

echo "ERROR: new version is not healthy." >&2
rm -f stallfunk-admin.new web.new.tar.gz # keep tool and web app matching the API that runs again
journalctl -u reiterhof-api.service -n 30 --no-pager >&2 || true
if [[ -f api.prev ]]; then
  echo "Rolling back to the previous binary." >&2
  cp -p api.prev api.rollback.tmp
  mv -f api.rollback.tmp api
  sudo systemctl restart reiterhof-api.service
  healthy && echo "Rollback healthy." >&2 || echo "ERROR: rollback is not healthy either!" >&2
fi
exit 1
