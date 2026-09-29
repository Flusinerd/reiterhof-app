# Reiterhof

Mobile App für die Stallgasse: Anwesenheit, Decken und Wetter, Anfragen an
Helfer, Pferdeakte und Training.

| Verzeichnis | Inhalt |
| --- | --- |
| `backend/` | Go-Service, Cloud-API. `net/http` ServeMux, kein Framework. |
| `mobile/` | Expo-/React-Native-App (TypeScript, Expo Router). |

## Loslegen

Voraussetzungen: Go 1.26, Node 22, [`just`](https://just.systems), Postgres 16.

```sh
just setup    # Abhängigkeiten installieren
just db-reset  # lokale DB "reiterhof" anlegen, migrieren, Beispieldaten laden
just api      # API auf :8080 starten (migriert beim Start)
just app      # Expo-Entwicklungsserver
just test     # Tests
```

Die API liest `REITERHOF_DATABASE_URL` (Standard:
`postgres://postgres:postgres@localhost:5432/reiterhof?sslmode=disable`).
Datenbanktests laufen nur mit `REITERHOF_TEST_DATABASE_URL` (Admin-URL, z. B.
`postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`), sonst werden sie übersprungen.
Aufbau des Backends, Migrationen und Konventionen: [`docs/architecture.md`](docs/architecture.md).

Die App erwartet die API unter `http://10.0.2.2:8080` (Android-Emulator).
Für ein echtes Gerät `expo.extra.apiUrl` in `mobile/app.json` setzen.

## Konventionen

- Abhängigkeiten sind exakt gepinnt, keine `^`- oder `~`-Bereiche.
- Go-Tests nur mit der Standardbibliothek.
- Code, Bezeichner und Code-Kommentare sind englisch; nur die UI-Texte sind deutsch.
