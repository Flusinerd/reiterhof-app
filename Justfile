# Reiterhof — Aufgaben für die lokale Entwicklung.
# Übersicht: `just` oder `just --list`

_default:
  @just --list

# Einmalige Einrichtung nach dem Klonen
setup:
  go work sync
  cd mobile && npm ci

# API lokal starten (Port 8080)
api:
  cd backend && go run ./cmd/api

# Expo-Entwicklungsserver starten
app:
  cd mobile && npm start

# Linter und Typprüfung
lint:
  cd backend && go vet ./...
  cd mobile && npm run typecheck

# Alle Tests
test:
  cd backend && go test ./...
  cd mobile && npm test
