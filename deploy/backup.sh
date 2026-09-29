#!/usr/bin/env bash
# Daily backup of the Reiterhof database and uploads.
#
#   1. pg_dump -Fc of the database into $BACKUP_DIR/daily
#   2. tar.gz of the uploads directory
#   3. on Sundays, a copy of both into $BACKUP_DIR/weekly
#   4. retention: keep 14 daily and 8 weekly sets
#   5. offsite copy with rclone to $REITERHOF_RCLONE_REMOTE
#
# Any failed step is logged and makes the script exit non-zero at the end; the
# remaining steps still run so that one problem does not stop the other backups.
#
# Environment (normally from /etc/reiterhof/api.env and backup.env via systemd):
#   REITERHOF_DATABASE_URL         required, connection URI passed to pg_dump
#   REITERHOF_RCLONE_REMOTE        required unless REITERHOF_SKIP_OFFSITE=1, e.g. "offsite:reiterhof"
#   REITERHOF_UPLOAD_DIR           default /var/lib/reiterhof/uploads
#   REITERHOF_BACKUP_DIR           default /var/backups/reiterhof
#   REITERHOF_KEEP_DAILY           default 14
#   REITERHOF_KEEP_WEEKLY          default 8
#   REITERHOF_OFFSITE_KEEP_DAYS    default 90 (older offsite files are deleted)
#   REITERHOF_SKIP_OFFSITE         set to 1 to skip the offsite copy (not recommended)
#   REITERHOF_BACKUP_PING_URL      optional URL pinged after full success (dead man's switch)
set -euo pipefail

BACKUP_DIR="${REITERHOF_BACKUP_DIR:-/var/backups/reiterhof}"
UPLOAD_DIR="${REITERHOF_UPLOAD_DIR:-/var/lib/reiterhof/uploads}"
KEEP_DAILY="${REITERHOF_KEEP_DAILY:-14}"
KEEP_WEEKLY="${REITERHOF_KEEP_WEEKLY:-8}"
OFFSITE_KEEP_DAYS="${REITERHOF_OFFSITE_KEEP_DAYS:-90}"
: "${REITERHOF_DATABASE_URL:?REITERHOF_DATABASE_URL is not set}"

stamp="$(date +%Y-%m-%d)"
dump_name="reiterhof-${stamp}.dump"
uploads_name="reiterhof-${stamp}-uploads.tar.gz"
failed=0

log() { printf '%s %s\n' "$(date -Is)" "$*"; }
err() {
  printf '%s ERROR: %s\n' "$(date -Is)" "$*" >&2
  failed=1
}

umask 077
mkdir -p "$BACKUP_DIR/daily" "$BACKUP_DIR/weekly"

# --- 1. Database -----------------------------------------------------------------
# Write to a temp name first so an interrupted dump never looks like a valid backup.
log "Dumping database"
if pg_dump --format=custom --no-owner --file="$BACKUP_DIR/daily/.${dump_name}.tmp" \
  "$REITERHOF_DATABASE_URL" &&
  pg_restore --list "$BACKUP_DIR/daily/.${dump_name}.tmp" >/dev/null &&
  mv "$BACKUP_DIR/daily/.${dump_name}.tmp" "$BACKUP_DIR/daily/${dump_name}"; then
  log "Database dump ok: ${dump_name}"
else
  rm -f "$BACKUP_DIR/daily/.${dump_name}.tmp"
  err "database dump failed"
fi

# --- 2. Uploads -------------------------------------------------------------------
log "Archiving uploads"
tar_rc=0
tar --create --gzip --file="$BACKUP_DIR/daily/.${uploads_name}.tmp" \
  --directory="$(dirname "$UPLOAD_DIR")" "$(basename "$UPLOAD_DIR")" || tar_rc=$?
# tar exits 1 if a file changed while it was read; that is fine for append-only uploads.
if ((tar_rc <= 1)) && gzip --test "$BACKUP_DIR/daily/.${uploads_name}.tmp"; then
  mv "$BACKUP_DIR/daily/.${uploads_name}.tmp" "$BACKUP_DIR/daily/${uploads_name}"
  log "Uploads archive ok: ${uploads_name}"
else
  rm -f "$BACKUP_DIR/daily/.${uploads_name}.tmp"
  err "uploads archive failed (tar exit code ${tar_rc})"
fi

# --- 3. Weekly copy (Sunday) --------------------------------------------------------
if [[ "$(date +%u)" == "7" ]]; then
  for f in "$dump_name" "$uploads_name"; do
    if [[ -f "$BACKUP_DIR/daily/$f" ]]; then
      cp -f "$BACKUP_DIR/daily/$f" "$BACKUP_DIR/weekly/$f" || err "weekly copy of $f failed"
    fi
  done
fi

# --- 4. Retention ---------------------------------------------------------------------
# prune <dir> <keep>: keeps the newest <keep> dates. File names contain the ISO date, so
# lexical order equals chronological order.
prune() {
  local dir="$1" keep="$2" stale
  stale="$(find "$dir" -maxdepth 1 -type f -name 'reiterhof-????-??-??*' -printf '%f\n' |
    sed -e 's/^reiterhof-\([0-9-]\{10\}\).*/\1/' | sort -u | head -n "-${keep}")"
  local d
  for d in $stale; do
    log "Pruning ${dir##*/}/${d}"
    rm -f "$dir/reiterhof-${d}.dump" "$dir/reiterhof-${d}-uploads.tar.gz"
  done
}
prune "$BACKUP_DIR/daily" "$KEEP_DAILY" || err "pruning daily backups failed"
prune "$BACKUP_DIR/weekly" "$KEEP_WEEKLY" || err "pruning weekly backups failed"

# --- 5. Offsite copy --------------------------------------------------------------------
if [[ "${REITERHOF_SKIP_OFFSITE:-0}" == "1" ]]; then
  log "Offsite copy skipped (REITERHOF_SKIP_OFFSITE=1)"
elif [[ -z "${REITERHOF_RCLONE_REMOTE:-}" ]]; then
  err "REITERHOF_RCLONE_REMOTE is not set; no offsite backup was made"
else
  export RCLONE_CONFIG="${RCLONE_CONFIG:-/var/lib/reiterhof/rclone/rclone.conf}"
  log "Copying to ${REITERHOF_RCLONE_REMOTE}"
  # copy (not sync): a wiped local directory must never delete the offsite copies.
  rclone copy --immutable "$BACKUP_DIR/daily" "${REITERHOF_RCLONE_REMOTE%/}/daily" ||
    err "rclone copy of daily backups failed"
  rclone copy --immutable "$BACKUP_DIR/weekly" "${REITERHOF_RCLONE_REMOTE%/}/weekly" ||
    err "rclone copy of weekly backups failed"
  rclone delete --min-age "${OFFSITE_KEEP_DAYS}d" "${REITERHOF_RCLONE_REMOTE%/}/daily" ||
    err "offsite retention (daily) failed"
  rclone delete --min-age "$((OFFSITE_KEEP_DAYS * 4))d" "${REITERHOF_RCLONE_REMOTE%/}/weekly" ||
    err "offsite retention (weekly) failed"
fi

# --- Result -------------------------------------------------------------------------------
if ((failed)); then
  log "Backup FAILED"
  exit 1
fi
log "Backup finished successfully"
if [[ -n "${REITERHOF_BACKUP_PING_URL:-}" ]]; then
  curl -fsS -m 10 --retry 3 -o /dev/null "$REITERHOF_BACKUP_PING_URL" || log "warning: ping failed"
fi
