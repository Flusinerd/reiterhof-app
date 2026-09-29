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

# Linters and type checks
lint:
  cd backend && go vet ./...
  cd mobile && npm run typecheck

# All tests
test:
  cd backend && go test ./...
  cd mobile && npm test

# Fail on unpinned package.json versions and GitHub Actions not pinned to a SHA
check-pins:
  scripts/check-pins.sh

# Lint the shell scripts (requires shellcheck)
lint-scripts:
  shellcheck scripts/*.sh deploy/*.sh
