#!/usr/bin/env bash
# Wrapper for the operator tool. provision.sh installs it as /usr/local/bin/stallfunk-admin.
#
#   stallfunk-admin user list          (as the admin user; sudo happens inside)
#   sudo stallfunk-admin user list     (works as well)
#
# The binary /opt/reiterhof/stallfunk-admin (installed by the Deploy workflow) runs as the
# service user reiterhof, because only that user and root may read the configuration
# /etc/reiterhof/api.env (root:reiterhof 0640) and open /opt/reiterhof (deploy:reiterhof 2750).
# Running as reiterhof instead of root keeps the tool as unprivileged as the API itself;
# it never needs more than the API's own database access. sudo resets the environment, so the
# settings come from the env file only, not from the calling shell.
set -euo pipefail

BIN=/opt/reiterhof/stallfunk-admin
ENV_FILE=/etc/reiterhof/api.env

if [[ "$(id -un)" == reiterhof ]]; then
  exec "$BIN" --env-file "$ENV_FILE" "$@"
fi

# The directory is not readable for other users, so the check has to run as reiterhof.
if ! sudo -u reiterhof -- test -x "$BIN"; then
  echo "ERROR: $BIN is missing or not executable." >&2
  echo "Run the GitHub Actions workflow \"Deploy\" once; it installs the tool next to the API." >&2
  exit 1
fi
exec sudo -u reiterhof -- "$BIN" --env-file "$ENV_FILE" "$@"
