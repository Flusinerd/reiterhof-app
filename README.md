# Reiterhof

Mobile App für die Stallgasse: Anwesenheit, Decken und Wetter, Anfragen an
Helfer, Pferdeakte und Training.

| Verzeichnis | Inhalt |
| --- | --- |
| `backend/` | Go-Service, Cloud-API. `net/http` ServeMux, kein Framework. |
| `mobile/` | Expo-/React-Native-App (TypeScript, Expo Router). |

## Funktionen

- **Anwesenheit**: „Bin da“ und „Bin weg“, wer gerade im Stall ist, optional automatisch per Geofence.
- **Decken und Wetter**: Deckenplan pro Pferd nach Regeln, Vorschlag für die Nacht aus dem DWD-Wetter, Erinnerung an die letzte Person im Stall.
- **Anfragen**: Helfer für Reiten, Führen, Turniere oder Tierarzttermine finden, auch als Serie, mit Kalendereintrag.
- **Pferdeakte**: Stammdaten, Reitbeteiligungen mit Rechten, Notfallkarte, Gesundheitstermine mit Erinnerungen, Dokumente.
- **Auffälligkeiten**: melden (auch dringend, mit Alarm an alle im Stall), beobachten, in einen Reha-Plan umwandeln.
- **Training**: „Was heute?“, Wochenplan, Übungsbibliothek, Einheiten eintragen oder aufzeichnen (GPS-Ausritt, Halle mit Gangarterkennung).
- **Reha-Plan**: Phasen mit „Heute erlaubt“, Abbruchkriterien und Kontrolltermin.
- **Datenschutz**: Einwilligungen pro Funktion, Datenexport, Konto löschen, Datenschutzerklärung und Impressum in der App.

Fachliche Details je Bereich: [`docs/domains/`](docs/domains/).

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
Anmeldung (E-Mail-Link, Google, Apple) einrichten: [`docs/auth-setup.md`](docs/auth-setup.md). Lokal: `REITERHOF_DEV_LOGIN=true just api`, dann `POST /api/v1/auth/dev-login {"email":"jan@example.org"}`.

Die App erwartet die API unter `http://10.0.2.2:8080` (Android-Emulator).
Für ein echtes Gerät `expo.extra.apiUrl` in `mobile/app.json` setzen.

## Konventionen

- Abhängigkeiten sind exakt gepinnt, keine `^`- oder `~`-Bereiche.
- Go-Tests nur mit der Standardbibliothek.
- UI der App: Komponenten aus `mobile/components/ui/`, siehe [docs/design-system.md](docs/design-system.md).
- Code, Bezeichner und Code-Kommentare sind englisch; nur die UI-Texte sind deutsch.

## Einrichtung durch den Betreiber

Einmalige Schritte, die nicht im Repository stehen:

1. **Server und Betrieb**: VPS, DNS, TLS, Backups und Restore nach [`deploy/README.md`](deploy/README.md) und [`deploy/restore.md`](deploy/restore.md).
2. **Anmeldung**: SMTP für die Anmelde-Links, Google- und Apple-Anmeldung nach [`docs/auth-setup.md`](docs/auth-setup.md). Den ersten Stall und den ersten Admin legt der Betreiber direkt in der Datenbank an (es gibt dafür keine API, siehe [`docs/architecture.md`](docs/architecture.md#authentication-and-roles)); weitere Mitglieder treten mit einem Einladungscode bei.
3. **Rechtstexte**: die Entwürfe [`docs/legal/datenschutz.md`](docs/legal/datenschutz.md) und [`docs/legal/impressum.md`](docs/legal/impressum.md) ausfüllen und rechtlich prüfen lassen, Verträge mit Dienstleistern nach [`docs/legal/avv-checkliste.md`](docs/legal/avv-checkliste.md). Danach `node scripts/gen-legal.mjs` ausführen (erzeugt die Texte der App, siehe [`docs/domains/privacy.md`](docs/domains/privacy.md)).
4. **Google-Maps-API-Key für Android**: `react-native-maps` zeigt auf Android Google Maps und braucht einen Key in `mobile/app.json` unter `expo.android.config.googleMaps.apiKey`, sonst bleibt die Karte der Ausritte leer (iOS nutzt Apple Maps). Details in [`docs/domains/tracking.md`](docs/domains/tracking.md#setup-and-open-points).
5. **EAS-Projekt für Push**: `eas init` ausführen und die `projectId` in `mobile/app.json` unter `expo.extra.eas.projectId` eintragen; ohne sie gibt es keinen Push-Token. Für Push auf Android zusätzlich Firebase (FCM) in EAS hinterlegen, für iOS einen Apple-Developer-Zugang. Optional `REITERHOF_EXPO_ACCESS_TOKEN` auf dem Server ([`docs/architecture.md`](docs/architecture.md#push)).
6. **Testgeräte**: GPS-Tracking, Hintergrundortung, Karte und Push laufen nicht in Expo Go; ein Development Build ist nötig (siehe [`docs/domains/tracking.md`](docs/domains/tracking.md)).

## CI und Deployment

- CI (`.github/workflows/ci.yml`): Backend (`go vet`, `go test` mit Postgres 16), Mobile (Typecheck, Tests, `npx expo install --check`), Pin-Check und ShellCheck.
- `just check-pins` prüft lokal, dass alle Abhängigkeiten exakt gepinnt und alle GitHub Actions auf einen Commit-SHA fixiert sind.
- Deployment auf einen einzelnen Linux-VPS (Netcup): Runbook in [`deploy/README.md`](deploy/README.md), Restore in [`deploy/restore.md`](deploy/restore.md).
