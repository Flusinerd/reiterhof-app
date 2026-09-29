# Reiterhof — local development tasks.
# Overview: `just` or `just --list`

_default:
  @just --list

# One-time setup after cloning
setup:
  go work sync
  cd mobile && npm ci

# Start the API locally (port 8080)
api:
  cd backend && go run ./cmd/api

# Start the Expo dev server
app:
  cd mobile && npm start

# Local Postgres admin connection (used by db-reset) and the app database it (re)creates.
# The API, migrate and seed read REITERHOF_DATABASE_URL (default: database "reiterhof").
db_admin_url := env("REITERHOF_ADMIN_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")

# Apply pending SQL migrations to REITERHOF_DATABASE_URL
migrate:
  cd backend && go run ./cmd/migrate

# Load the example data (idempotent, migrates first)
seed:
  cd backend && go run ./cmd/seed

# Drop and recreate the local database "reiterhof", then migrate and seed (needs psql)
db-reset:
  psql "{{db_admin_url}}" -v ON_ERROR_STOP=1 -c 'DROP DATABASE IF EXISTS reiterhof WITH (FORCE)' -c 'CREATE DATABASE reiterhof'
  just migrate
  just seed

# Linters and type checks
lint:
  cd backend && go vet ./...
  cd mobile && npm run typecheck

# All tests. DB tests run when REITERHOF_TEST_DATABASE_URL is set in the environment
# (admin URL, e.g. postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable), else they skip.
test:
  cd backend && go test ./...
  cd mobile && npm test

# Fail on unpinned package.json versions and GitHub Actions not pinned to a SHA
check-pins:
  scripts/check-pins.sh

# Lint the shell scripts (requires shellcheck)
lint-scripts:
  shellcheck scripts/*.sh deploy/*.sh
