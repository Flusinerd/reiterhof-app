# Web-App (PWA)

Ticket: JAN-75. Stallfunk läuft als installierbare Web-App im Browser (Ziel: iPhone-Safari), gebaut aus
derselben Expo-Codebasis wie die native App. Die native App bleibt unverändert; wo native Module im
Browser nicht funktionieren, gibt es `*.web.ts(x)`-Varianten (Metro wählt sie beim Web-Build).

## Installieren (iPhone)

1. Die Adresse (`https://stallfunk.de`) in **Safari** öffnen und anmelden.
2. **Teilen** (Quadrat mit Pfeil) → **„Zum Home-Bildschirm“** → **„Hinzufügen“**.
3. Stallfunk über das Symbol auf dem Home-Bildschirm starten. Die App öffnet ohne Safari-Leisten.

Voraussetzungen und Grenzen:

- **iOS 16.4 oder neuer** für Web-Push (nur für installierte Web-Apps; siehe die Push-Doku des Web-Push-Tickets).
  Installieren und alles andere geht ab iOS 15. Nur Safari kann auf iOS installieren (Chrome/Firefox auf
  iOS verwenden Safari-Technik, bieten „Zum Home-Bildschirm“ aber ebenfalls an, ab iOS 16.4).
- Die installierte Web-App hat auf iOS einen **eigenen Speicher**, getrennt von Safari: nach dem Installieren
  muss man sich einmal neu anmelden.
- Safari löscht Website-Daten (localStorage, IndexedDB) nach sieben Tagen ohne Nutzung, **außer** bei
  installierten Web-Apps. Ein weiterer Grund zu installieren.
- Standort wird pro Start der Web-App neu abgefragt (Verhalten von iOS), Bewegungssensoren ebenfalls.

## Was geht, was nicht

| Funktion | Web | Umsetzung |
| --- | --- | --- |
| Anmelden (Code/Link, Google) | ja | Auth-Screens unverändert |
| „Mit Apple anmelden“ | **nein**, Button ausgeblendet | `capabilities.appleSignIn` in `components/social-sign-in.tsx` |
| Start, Decken, Anfragen, Training, Pferde, Auffälligkeiten, Reha, Erinnerungen | ja | gemeinsame Screens |
| Echtzeit-Updates (SSE) | ja | `lib/realtime.ts` mit `expo/fetch`, gegen `API_URL` |
| Anwesenheit „Bin da / Ich gehe“ | ja | |
| Automatisches Ein-/Auschecken (Geofence) | **nein**, Hinweistext „Automatisch ein- und auschecken gibt es nur in der App aus dem App Store.“ | `lib/geofence.web.ts` (Stub „unsupported“) |
| GPS-Aufzeichnung | nur **im Vordergrund** | `lib/tracking-location.web.ts` (`watchPosition`), Wake Lock, Warnung „Bildschirm während der Aufzeichnung anlassen“ |
| Gangarterkennung | ja, wenn der Browser Bewegungsdaten liefert | `lib/tracking-sensors.web.ts` (`devicemotion`, m/s² → g) |
| Karte der Ausritte | ja | `components/tracking-map.web.tsx` (MapLibre GL JS, OpenFreeMap) |
| Fotos und Dokumente hochladen | ja | `expo-image-picker` / `expo-document-picker` (Web-Dateiauswahl), `lib/upload.ts` |
| Kalendereintrag | Datei statt Kalender-Dialog | `lib/requests-calendar.web.ts`: ICS über Web Share (iOS-Teilen-Menü → „Zum Kalender“), sonst Download |
| Datenexport (Datenschutz) | ja, als Datei | `lib/export-file.web.ts` |
| Bestätigungsdialoge („Löschen?“) | ja | `Alert.alert` gibt es in react-native-web nicht; `lib/web-setup.web.ts` leitet auf `window.confirm` um |
| Push-Benachrichtigungen | separat (Web-Push-Ticket) | `lib/push.web.ts`, `public/sw.js` |

## Architektur

- **Ausgabe**: `expo.web.output = "single"` (SPA). Der Host liefert `index.html` für unbekannte Pfade.
  `static` lohnt hier nicht: alle Screens brauchen den angemeldeten Zustand, es gäbe nichts zu prerendern, und
  jede Route würde als eigene Datei ausgeliefert. **Wichtig:** Bei `single` liest Expo `app/+html.tsx`
  nicht; die HTML-Hülle (Viewport mit `viewport-fit=cover`, `apple-mobile-web-app-*`, Manifest, Icons,
  Hintergrundfarbe als „Splash“) ist `mobile/public/index.html`.
- **Manifest**: `mobile/public/manifest.webmanifest` (`display: standalone`, `start_url`/`scope` `/`, Farben aus
  den Design-Tokens, `lang: de`, Icons any + maskable). Icons erzeugt `node scripts/gen-icons.mjs`
  (reines JS, ein „S“ in `#2d5a3d` auf `#f6f4ee`); `npm run icons`.
- **API-Adresse**: `lib/api.ts` → `resolveApiUrl` (`lib/platform-core.ts`). Web: `window.location.origin`
  (Caddy leitet `/api` an das Backend). Überschreiben beim Build mit `STALLFUNK_API_URL` (über `app.config.js`)
  oder `EXPO_PUBLIC_API_URL`, z. B. `STALLFUNK_API_URL=http://localhost:8080 npx expo export --platform web`.
  Realtime, Dateiadressen (`?access_token=`, `fileSource`) und ICS-Links nutzen dieselbe Basis.
- **Speicher**: `lib/storage.ts` (SecureStore) bzw. `lib/storage.web.ts` (**localStorage**) für Sitzungs-Token,
  Geofence-Schalter, „Einwilligung schon gefragt“. localStorage ist schwächer geschützt als der
  Schlüsselbund: jedes Skript der Origin kann es lesen. Deshalb keine Drittanbieter-Skripte und eine
  strenge CSP auf dem Host; das Token ist ein widerrufbares Bearer-Token. Laufende Ausritte
  (`lib/tracking-files.web.ts`) liegen in **IndexedDB** (Hunderte KB, localStorage hat etwa 5 MB).
- **Plattformfähigkeiten**: `lib/platform.ts` (`isWeb`, `capabilities`), Logik in `lib/platform-core.ts`
  (getestet). Screens fragen Fähigkeiten, nicht `Platform.OS`.
- **Bewegungssensor auf iOS**: `DeviceMotionEvent.requestPermission()` muss aus einer Berührung heraus aufgerufen
  werden. `useTrackingSession.begin/resume` rufen sie als Erstes auf. Kommt zwischen Tippen und Aufruf ein
  Einwilligungs-Dialog (erste GPS-Aufzeichnung), kann iOS die Abfrage ablehnen; dann zeigt die App „Kein
  Bewegungssensor gefunden“ und schätzt die Gangart aus dem GPS-Tempo. Unverifiziert auf einem echten iPhone.
- **Karte**: MapLibre GL JS (`maplibre-gl` 6.11.2, exakt gepinnt), Stil OpenFreeMap „positron“ (kostenlos, ohne
  Schlüssel, Attribution wird angezeigt). Die OSM-Standardserver sind für regelmäßigen App-Verkehr nicht erlaubt
  ([Tile Usage Policy](https://operations.osmfoundation.org/policies/tiles/)). Für viele Nutzer selbst hosten oder
  `EXPO_PUBLIC_MAP_STYLE_URL` setzen. Der Worker (`maplibre-gl-worker.mjs`) wird per `postinstall`
  (`scripts/copy-maplibre-worker.mjs`) nach `public/maplibre/` kopiert (git-ignoriert), weil MapLibre die
  Worker-Adresse sonst aus `import.meta.url` ableitet.

## Hosting (JAN-77)

Produktion: `https://stallfunk.de`, derselbe VPS wie die API (Runbook: [`deploy/README.md`](../../deploy/README.md#web-app-pwa)).

- **Caddy** (`deploy/Caddyfile`, Host aus `REITERHOF_WEB_DOMAIN` in `/etc/reiterhof/caddy.env`): `/api/*` und
  `/auth/verify` gehen an die API (`127.0.0.1:8080`, gleiche Einstellungen wie der API-Host: Upload-Limit 25 MB,
  Health-Check; der Echtzeit-Stream `text/event-stream` wird nicht gepuffert und nicht komprimiert). Alles
  andere kommt aus `/var/www/stallfunk/current`. `api.stallfunk.de` bleibt unverändert für die native App.
- **SPA-Fallback**: Pfade ohne Dateiendung und ohne Datei (`/horses`, `/requests/12`) liefern `index.html`;
  `/_expo/*`, `/maplibre/*`, `/assets/*` und alles mit Endung liefern bei fehlender Datei **404**.
- **Cache**: `/_expo/static/*` und `/assets/*` (Hash im Namen) `public, max-age=31536000, immutable`; alles andere
  (`index.html`, `sw.js`, `manifest.webmanifest`, Icons, MapLibre-Worker) `no-cache`. Fehlerantworten `no-store`.
- **MIME**: `.mjs` und `.js` als `text/javascript` (Caddy-Standard), `manifest.webmanifest` wird vom Caddyfile als
  `application/manifest+json` gesetzt (Caddy liefert sonst `text/plain`).
- **Header** (Web-Host): HSTS, `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`; CSP
  `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' https://tiles.openfreemap.org; worker-src 'self' blob:; font-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'`
  und `Permissions-Policy: camera=(), microphone=(), geolocation=(self), accelerometer=(self), gyroscope=(self), screen-wake-lock=(self)`.
  Für die proxied Pfade gilt als Default die CSP des API-Hosts (`default-src 'none'`; `?Content-Security-Policy`,
  d. h. eine vom Backend gesetzte CSP bleibt stehen, siehe Login-Mail). Das `index.html` hat kein
  Inline-Skript; die Google-Anmeldung öffnet `accounts.google.com` in einem Popup (von der CSP nicht betroffen).
  Wer die Kartenquelle wechselt (`EXPO_PUBLIC_MAP_STYLE_URL`), muss `connect-src` anpassen.
- **Deploy**: `deploy.yml` baut den Export (Node 22, `npm ci`, `npx expo export --platform web`), lädt
  `web.tar.gz` hoch, `remote-swap.sh` entpackt nach `/var/www/stallfunk/releases/<sha>` und schaltet den Symlink
  `current` atomar um (nach dem API-Health-Check, letzte 3 Releases bleiben). CI baut den Export ebenfalls.
- **Betreiber-Checkliste**: DNS (`stallfunk.de` A/AAAA, `api.stallfunk.de` CNAME), einmalig `provision.sh` mit
  `REITERHOF_WEB_DOMAIN=stallfunk.de` erneut ausführen, Deploy starten. Für Web-Push das VAPID-Paar
  (`REITERHOF_VAPID_PUBLIC_KEY`, `REITERHOF_VAPID_PRIVATE_KEY`, `REITERHOF_VAPID_SUBJECT`) in `api.env` setzen
  ([push-web.md](push-web.md)); ohne Schlüssel antwortet `/api/v1/push/web/public-key` mit 503 und Web-Push ist aus.
- **Login-Mail**: `/auth/verify?token=…` (falls `REITERHOF_PUBLIC_URL` gesetzt ist) zeigt die Seite der API mit dem
  Link `stallfunk://…`; sie meldet nicht in der Web-App an. Die Seite ist im Stallfunk-Design (Systemschriften, Logo
  als `data:`-URI) und setzt ihre eigene CSP (`style-src` mit SHA-256-Hash des Stylesheets, `img-src data:`);
  Caddy setzt `default-src 'none'` nur, wenn das Backend keine CSP mitschickt.

## Anforderungen an den Host

- `index.html` für alle unbekannten Pfade (SPA-Fallback), aber **nicht** für Dateien mit Endung
  (`/_expo/…`, `/maplibre/…`, `/manifest.webmanifest`, Icons) und nicht für `/api`.
- Cache: `/_expo/static/**` unveränderlich (`Cache-Control: public, max-age=31536000, immutable`, Hashes im Namen);
  `index.html`, `manifest.webmanifest`, `sw.js` mit `no-cache`.
- MIME-Typen: `manifest.webmanifest` als `application/manifest+json`, **`.mjs` als `text/javascript`** (sonst
  startet der Karten-Worker nicht), `.js` als `text/javascript`.
- CSP (wenn gesetzt): `connect-src 'self' https://tiles.openfreemap.org`, `img-src 'self' data: blob:`,
  `worker-src 'self' blob:`, `style-src 'self' 'unsafe-inline'`.
- HTTPS ist Pflicht (Standort, Bewegungssensoren, Service Worker, Web Share).

## Lokal ausprobieren

```sh
just api                                   # oder REITERHOF_DEV_LOGIN=true …
cd mobile
STALLFUNK_API_URL=http://localhost:8080 npx expo start --web   # Dev-Server
# oder statisch:
STALLFUNK_API_URL=http://localhost:8080 npx expo export --platform web --output-dir /tmp/web
```

`STALLFUNK_API_URL` zeigt beim Export auf einen anderen Ursprung als die Seite; das Backend muss dann CORS
erlauben. Einfacher: einen kleinen Server, der `dist/` ausliefert und `/api` an `:8080` weiterreicht
(wie Caddy in Produktion), und `STALLFUNK_API_URL` nicht setzen.

## Offen

- Kein Prüfen auf echtem iPhone (Wake Lock, Bewegungssensor-Berechtigung, Teilen-Menü für ICS).
- iOS-Startbilder (`apple-touch-startup-image`) fehlen: beim Start zeigt iOS die Hintergrundfarbe der Seite.
- Der Einwilligungstext für Push nennt Expo, Apple und Google; für Web-Push ist er anzupassen.
- Service Worker: `lib/web-setup.web.ts` enthält den Platz, an dem `registerServiceWorker()` aufzurufen ist.
