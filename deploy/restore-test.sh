#!/usr/bin/env bash
# Verifies that a backup can actually be restored, without touching the live data.
#
#   sudo restore-test.sh /var/backups/reiterhof/daily/reiterhof-2026-01-31.dump \
#                        [/var/backups/reiterhof/daily/reiterhof-2026-01-31-uploads.tar.gz]
#
# Restores the dump into a throw-away database (reiterhof_restoretest), prints row
# counts of all user tables, unpacks the uploads archive into a temp directory and
# cleans everything up. Exits non-zero if anything fails or the dump has no tables.
#
# Run as root; PostgreSQL commands are executed as the "postgres" OS user. The dump is
# fed through stdin so it does not have to be readable by that user.
# Connection settings can be overridden with the usual PGHOST/PGPORT variables.
set -euo pipefail

dump="${1:?usage: $0 <dump-file> [uploads-tar.gz]}"
uploads="${2:-}"
db="reiterhof_restoretest"
workdir="$(mktemp -d)"

[[ $EUID -eq 0 ]] || { echo "ERROR: run as root" >&2; exit 1; }
[[ -r "$dump" ]] || { echo "ERROR: cannot read $dump" >&2; exit 1; }

as_pg() { runuser -u postgres -- "$@"; }
cleanup() {
  as_pg dropdb --if-exists "$db" >/dev/null 2>&1 || true
  rm -rf "$workdir"
}
trap cleanup EXIT

echo "==> Restoring $dump into scratch database $db"
as_pg dropdb --if-exists "$db"
as_pg createdb "$db"
as_pg pg_restore --exit-on-error --no-owner --dbname "$db" <"$dump"

echo "==> Row counts"
# Counts every table outside the system schemas; prints "schema.table<TAB>rows".
counts="$(as_pg psql -X -At -v ON_ERROR_STOP=1 -d "$db" <<'SQL'
SELECT format('SELECT %L, count(*) FROM %I.%I', schemaname || '.' || tablename, schemaname, tablename)
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY 1 \gexec
SQL
)"
if [[ -z "$counts" ]]; then
  echo "ERROR: the restored database contains no tables" >&2
  exit 1
fi
printf '%s\n' "$counts" | tr '|' '\t'

if [[ -n "$uploads" ]]; then
  echo "==> Unpacking $uploads"
  [[ -r "$uploads" ]] || { echo "ERROR: cannot read $uploads" >&2; exit 1; }
  tar --extract --gzip --file="$uploads" --directory="$workdir"
  echo "Files in uploads archive: $(find "$workdir" -type f | wc -l)"
fi

echo "==> Restore test passed"
