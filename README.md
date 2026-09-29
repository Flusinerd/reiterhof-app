# Reiterhof

Mobile App für die Stallgasse: Anwesenheit, Decken und Wetter, Anfragen an
Helfer, Pferdeakte und Training.

| Verzeichnis | Inhalt |
| --- | --- |
| `backend/` | Go-Service, Cloud-API. `net/http` ServeMux, kein Framework. |
| `mobile/` | Expo-/React-Native-App (TypeScript, Expo Router). |

## Loslegen

Voraussetzungen: Go 1.26, Node 22, [`just`](https://just.systems).

```sh
just setup   # Abhängigkeiten installieren
just api     # API auf :8080 starten
just app     # Expo-Entwicklungsserver
just test    # Tests
```

Die App erwartet die API unter `http://10.0.2.2:8080` (Android-Emulator).
Für ein echtes Gerät `expo.extra.apiUrl` in `mobile/app.json` setzen.

## Konventionen

- Abhängigkeiten sind exakt gepinnt, keine `^`- oder `~`-Bereiche.
- Go-Tests nur mit der Standardbibliothek.
- UI der App: Komponenten aus `mobile/components/ui/`, siehe [docs/design-system.md](docs/design-system.md).
- Code, Bezeichner und Code-Kommentare sind englisch; nur die UI-Texte sind deutsch.
