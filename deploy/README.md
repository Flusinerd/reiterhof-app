# Deployment auf einen Netcup-VPS

Runbook für den Betrieb der Stallfunk-API und der Web-App (PWA) auf **einem** Debian-/Ubuntu-Server
(Netcup-VPS mit 4 GB RAM). Alles läuft auf dem Server:

```
Internet -> Caddy (80/443, automatisches TLS) -> reiterhof-api (127.0.0.1:8080) -> PostgreSQL 16 (localhost)
             |-- api.stallfunk.de   nur die API (native App)
             `-- stallfunk.de       Web-App aus /var/www/stallfunk/current, /api/* an dieselbe API
```

| Datei | Zweck |
| --- | --- |
| `provision.sh` | Einmalige, wiederholbare Server-Einrichtung (als root) |
| `Caddyfile` | Reverse Proxy und Web-App-Hosting, TLS, Security-Header (inkl. CSP der Web-App; für die API nur ein Default, `/auth/verify` und `/parental-consent` setzen ihre eigene), Upload-Limit 25 MB |
| `postgresql.conf.d/reiterhof.conf` | PostgreSQL-Tuning für 4 GB RAM |
| `systemd/reiterhof-api.service` | API als gehärteter systemd-Dienst |
| `api.env.example` | Vorlage für `/etc/reiterhof/api.env` |
| `backup.sh`, `systemd/reiterhof-backup.*` | Tägliches Backup + Offsite-Kopie |
| `restore-test.sh`, `restore.md` | Restore-Anleitung und Restore-Test |
| `remote-swap.sh` | Wird vom Deploy-Workflow auf dem Server ausgeführt (API, Betriebswerkzeug, Web-App) |
| `stallfunk-admin.sh` | Wrapper `/usr/local/bin/stallfunk-admin` für das Betriebswerkzeug (siehe [Betrieb](#betrieb-stallfunk-admin)) |

## 1. Server bestellen

1. Bei Netcup einen VPS mit 4 GB RAM bestellen (z. B. VPS Lite 1: 2 vCore, 4 GB RAM, 80 GB SSD) (Rechenzentrum in Deutschland,
   damit die Daten in der EU bleiben).
2. Image: **Debian 13 (Minimal)** empfohlen; Debian 12 und Ubuntu 24.04 funktionieren ebenfalls. Beim Anlegen deinen SSH-Public-Key
   hinterlegen (Server Control Panel, SCP). Beide Distributionen werden unterstützt.
3. Die IPv4-/IPv6-Adresse des Servers notieren.

## 2. DNS

Bei deinem DNS-Anbieter für die Web-Domain (z. B. `reiterhof.example`) anlegen:

- `A`-Record auf die IPv4-Adresse,
- `AAAA`-Record auf die IPv6-Adresse (optional, aber empfohlen).

Die API-Domain (z. B. `api.reiterhof.example`, für die native App) bekommt entweder dieselben
Einträge oder einen `CNAME` auf die Web-Domain (so ist es bei `stallfunk.de` eingerichtet).

Prüfen: `dig +short reiterhof.example api.reiterhof.example`. Ohne korrekten DNS-Eintrag kann Caddy
kein Zertifikat bekommen (je Domain eines). Ports 80 und 443 müssen erreichbar sein (die Firewall wird von
`provision.sh` gesetzt; in der Netcup-SCP-Firewall darf nichts blockieren).

## 3. Server einrichten (`provision.sh`)

Zwei SSH-Schlüssel werden gebraucht: einer für dich (Benutzer `admin`, volle sudo-Rechte)
und ein **separater** für GitHub Actions (Benutzer `deploy`, darf nur die API neu starten).
Deploy-Schlüssel lokal erzeugen (ohne Passphrase, nur für diesen Zweck):

```sh
ssh-keygen -t ed25519 -N '' -C reiterhof-deploy -f ./reiterhof-deploy
```

Dann `deploy/` auf den Server kopieren und als root ausführen:

```sh
scp -r deploy root@<server-ip>:/root/reiterhof-deploy
ssh root@<server-ip>
cd /root/reiterhof-deploy
REITERHOF_DOMAIN=api.reiterhof.example \
REITERHOF_WEB_DOMAIN=reiterhof.example \
REITERHOF_DB_PASSWORD="$(openssl rand -hex 24)" \
REITERHOF_ADMIN_SSH_PUBKEY="$(head -n1 /root/.ssh/authorized_keys)" \
REITERHOF_DEPLOY_SSH_PUBKEY="ssh-ed25519 AAAA... reiterhof-deploy" \
./provision.sh
```

Das Skript erledigt:

- Systembenutzer `reiterhof` (ohne Login), Login-Benutzer `admin` und `deploy`
- SSH-Härtung: kein Root-Login, keine Passwort-Anmeldung
- `ufw` (22 mit Rate-Limit, 80, 443), automatische Sicherheitsupdates (Neustart bei Bedarf um 04:30)
- PostgreSQL 16 (PGDG-Repository) inkl. Tuning, Datenbank und Rolle
- Caddy (offizielles Repository), systemd-Units, Verzeichnisse `/opt/reiterhof`,
  `/var/www/stallfunk` (Web-App, gehört dem Benutzer `deploy`, Caddy liest nur),
  `/var/lib/reiterhof/uploads`, `/var/backups/reiterhof`
- `/etc/reiterhof/caddy.env` mit `REITERHOF_DOMAIN` (API) und `REITERHOF_WEB_DOMAIN` (Web-App)
- `/etc/reiterhof/api.env` (nur wenn noch nicht vorhanden, mit dem DB-Passwort)

Wichtig: **Vor dem Schließen der Root-Sitzung** in einem neuen Terminal testen, dass
`ssh admin@<server-ip>` und `ssh -i reiterhof-deploy deploy@<server-ip>` funktionieren.
Das Skript ist idempotent und kann nach Änderungen erneut laufen.

## 4. Konfiguration prüfen

Auf dem Server als `admin`:

```sh
sudo nano /etc/reiterhof/api.env      # Vorlage: deploy/api.env.example
```

Die Datenbank-URL ist bereits eingetragen. Weitere Werte kommen hinzu, sobald die API sie
kennt. Passwörter nie ins Repository committen.

### Push für die native App (APNs und FCM)

Die API sendet direkt an Apple und Google, ohne Expo-Dienst dazwischen. Die Zugangsdaten sind zwei
Dateien neben `api.env` (Herkunft: README, „Einrichtung durch den Betreiber“, Punkt 5):

```bash
sudo install -m 0640 -o root -g reiterhof AuthKey_ABC123DEFG.p8 /etc/reiterhof/apns.p8
sudo install -m 0640 -o root -g reiterhof stallfunk-firebase-adminsdk.json /etc/reiterhof/fcm-service-account.json
sudo nano /etc/reiterhof/api.env      # REITERHOF_APNS_* und REITERHOF_FCM_SERVICE_ACCOUNT_FILE, Vorlage: api.env.example
sudo systemctl restart reiterhof-api  # im Journal: "native push" platforms=[ios android]
```

Fehlt eine Plattform, überspringt die API deren Geräte beim Senden (Logzeile), die andere läuft. Ein
abgelehnter Schlüssel steht beim Start als Fehler im Journal. Die Dateien sind Geheimnisse wie das
DB-Passwort: nicht ins Repository, nicht ins Backup außerhalb des Servers, bei Verdacht im Apple- bzw.
Firebase-Konto widerrufen und neu erzeugen.

### Web-Push (PWA auf dem iPhone)

Damit die Web-App Mitteilungen zustellen kann (iOS 16.4 oder neuer, Seite über Safari zum
Home-Bildschirm hinzugefügt), braucht die API ein VAPID-Schlüsselpaar. Ohne Schlüssel ist Web-Push
aus (`GET /api/v1/push/web/public-key` antwortet `503 not_configured`), die native App ist nicht betroffen.

1. Paar einmalig erzeugen, auf einem Rechner mit Go im Repository:
   `cd backend && go run ./cmd/vapidkeys -subject mailto:du@example.org`

   Oder ohne Go direkt auf dem Server mit `openssl` (gibt die drei Zeilen aus):

   ```sh
   k=$(mktemp) && openssl ecparam -name prime256v1 -genkey -noout -out "$k" &&
   echo "REITERHOF_VAPID_PUBLIC_KEY=$(openssl ec -in "$k" -pubout -outform DER 2>/dev/null | tail -c 65 | base64 -w0 | tr '/+' '_-' | tr -d '=')" &&
   echo "REITERHOF_VAPID_PRIVATE_KEY=$(openssl ec -in "$k" -outform DER 2>/dev/null | tail -c +8 | head -c 32 | base64 -w0 | tr '/+' '_-' | tr -d '=')" &&
   echo "REITERHOF_VAPID_SUBJECT=mailto:du@example.org"; rm -f "$k"
   ```
2. Die drei ausgegebenen Zeilen in `/etc/reiterhof/api.env` eintragen (`REITERHOF_VAPID_PUBLIC_KEY`,
   `REITERHOF_VAPID_PRIVATE_KEY`, `REITERHOF_VAPID_SUBJECT`; Vorlage in `deploy/api.env.example`).
   Der Betreff ist die Kontaktadresse (`mailto:` oder `https://`), an die sich Apple und Google bei
   Problemen wenden.
3. `sudo systemctl restart reiterhof-api`. Beim Start steht im Journal, ob Web-Push aktiv ist.

Den privaten Schlüssel nicht ins Repository committen und nicht mehr austauschen: Alle bestehenden
Browser-Abos würden ungültig, und jede Person müsste Mitteilungen neu aktivieren. Details:
[`docs/domains/push-web.md`](../docs/domains/push-web.md).

## 5. Erstes Deployment

Im GitHub-Repository unter *Settings > Secrets and variables > Actions* (am besten als
Environment `production`, dort lassen sich auch Freigaben verlangen):

| Secret | Inhalt |
| --- | --- |
| `DEPLOY_HOST` | Hostname oder IP des Servers |
| `DEPLOY_USER` | `deploy` |
| `DEPLOY_SSH_KEY` | privater Schlüssel `reiterhof-deploy` (komplette Datei) |
| `DEPLOY_KNOWN_HOSTS` | Ausgabe von `ssh-keyscan -t ed25519 <DEPLOY_HOST>` (Fingerprint vorher über das Netcup-SCP-Konsolenfenster prüfen) |

Optionale Variablen: `DEPLOY_PUBLIC_URL` (z. B. `https://api.reiterhof.example`, aktiviert den
externen Health-Check am Ende), `DEPLOY_WEB_URL` (z. B. `https://reiterhof.example`, prüft am Ende,
dass Web-App und Deep Link ausgeliefert werden) und `DEPLOY_ENABLED=true` (automatisches Deployment, sobald die CI
für einen Push auf `main` grün ist; ausgerollt wird genau der getestete Commit. Ohne diese
Variable läuft der Workflow nur manuell).

Dann *Actions > Deploy > Run workflow*. Der Workflow testet, baut zwei statische
linux/amd64-Binaries (API und das Betriebswerkzeug `stallfunk-admin`) sowie den Web-Export der App
(`npx expo export --platform web`, Node 22), lädt alles per `scp` hoch,
tauscht `/opt/reiterhof/api` atomar aus, startet den Dienst neu und prüft `/healthz`. Fällt der
Health-Check durch, wird automatisch das vorherige Binary wiederhergestellt
(`/opt/reiterhof/api.prev`). Das Betriebswerkzeug wird erst nach erfolgreichem Health-Check als
`/opt/reiterhof/stallfunk-admin` installiert (ebenfalls atomar, ohne zusätzliche sudo-Rechte).
Ebenfalls erst danach kommt die Web-App: siehe [Web-App (PWA)](#web-app-pwa).

Kontrolle auf dem Server:

```sh
systemctl status reiterhof-api
journalctl -u reiterhof-api -f
curl -fsS https://api.reiterhof.example/healthz
```

Datenbank-Migrationen laufen beim Start der API automatisch.

## Web-App (PWA)

Die Web-App (Expo-Web-Export, Single-Page-App) läuft unter der Web-Domain (`https://stallfunk.de`).
Caddy liefert dort die statischen Dateien aus `/var/www/stallfunk/current` und reicht `/api/*` (inkl.
Echtzeit-Stream `/api/v1/events`) sowie `/auth/verify` an dieselbe API weiter; die Web-App ruft die API
unter ihrer eigenen Adresse auf (gleicher Ursprung, kein CORS). Die native App nutzt weiter
`https://api.stallfunk.de`. Die Web-Domain hat ein eigenes Zertifikat (automatisch).

Ablauf pro Deployment (in `remote-swap.sh`, erst nach gesundem API-Health-Check):

1. Der Workflow lädt `web.tar.gz` nach `/opt/reiterhof/web.new.tar.gz` hoch.
2. Das Archiv wird nach `/var/www/stallfunk/releases/<commit-sha>` entpackt.
3. Der Symlink `/var/www/stallfunk/current` wird atomar umgeschaltet (`ln -sfn` + `mv -T`).
4. Die letzten 3 Releases bleiben liegen, ältere werden gelöscht.

Fällt der Health-Check der API durch, wird die Web-App nicht ausgetauscht.

**Einmalig auf dem bestehenden Server** (bevor der erste Deploy mit Web-App läuft), als `admin`:

1. DNS prüfen: `dig +short stallfunk.de` zeigt die Server-IP, `api.stallfunk.de` ist ein `CNAME` auf
   `stallfunk.de` (oder zeigt ebenfalls auf den Server).
2. Repository auf dem Server aktualisieren und `provision.sh` erneut ausführen, jetzt mit der neuen
   Variable `REITERHOF_WEB_DOMAIN` (Befehl unter [Betrieb](#betrieb-stallfunk-admin), „Einmalig auf
   dem bestehenden Server nachrüsten“). Das legt `/var/www/stallfunk` an, schreibt die Web-Domain in
   `/etc/reiterhof/caddy.env`, installiert das neue Caddyfile und startet Caddy neu. Danach steht
   `https://stallfunk.de` mit 404, bis der erste Deploy die Web-App liefert.
3. Optional die Repository-Variable `DEPLOY_WEB_URL=https://stallfunk.de` setzen (Deploy prüft dann die Web-App).
4. *Actions > Deploy > Run workflow* (oder der nächste Merge auf `main` mit `DEPLOY_ENABLED=true`).
5. Für Web-Push das VAPID-Schlüsselpaar in `api.env` eintragen (siehe [Web-Push](#web-push-pwa-auf-dem-iphone)),
   sonst bleibt Web-Push aus.
6. Optional in `api.env` `REITERHOF_PUBLIC_URL` prüfen: Login-Mails enthalten `<url>/auth/verify?token=...`,
   das eine Seite mit dem Link in die native App zeigt. Beide Domains liefern diese Seite aus.

**Prüfen**

```sh
curl -I https://stallfunk.de/sw.js                 # 200, Content-Type text/javascript, Cache-Control: no-cache
curl -I https://stallfunk.de/manifest.webmanifest  # Content-Type application/manifest+json
curl -I https://stallfunk.de/horses                # 200 (Deep Link liefert index.html)
curl -I https://stallfunk.de/gibt-es-nicht.js      # 404 (kein index.html für fehlende Dateien)
curl -I https://stallfunk.de/api/v1/me                # 401 (Antwort kommt von der API, nicht index.html)
curl -fsS https://api.stallfunk.de/healthz       # API-Host unverändert
ls -l /var/www/stallfunk/ /var/www/stallfunk/releases/
```

Danach auf dem iPhone in Safari `https://stallfunk.de` öffnen, „Zum Home-Bildschirm“ hinzufügen, anmelden
und die Mitteilungen erlauben (Anleitung und Grenzen: [`docs/domains/pwa.md`](../docs/domains/pwa.md)).

**Rollback der Web-App** (ohne neuen Deploy), auf dem Server als `deploy` (oder `admin`):

```sh
ls -t /var/www/stallfunk/releases/                # neuestes zuerst
cd /var/www/stallfunk
ln -sfn /var/www/stallfunk/releases/<alter-sha> current.new && mv -T current.new current
```

Der nächste Deploy schaltet wieder auf den neuen Stand um. Die Web-App ist eine Single-Page-App mit
Service Worker ohne Cache-Schicht: Browser holen `index.html` bei jedem Start neu (`no-cache`), die
Dateien unter `/_expo/static/` tragen einen Hash im Namen und werden ein Jahr gecacht.

## 6. Backups

Der Timer `reiterhof-backup.timer` läuft täglich gegen 03:00 Uhr: `pg_dump -Fc` und ein
Archiv der Uploads nach `/var/backups/reiterhof` (14 tägliche + 8 wöchentliche Stände),
danach Kopie per `rclone` an einen externen Speicher.

1. Offsite-Ziel wählen (z. B. Hetzner Storage Box per SFTP, S3-kompatibler Speicher eines EU-Anbieters).
   Nicht beim selben Anbieter/Rechenzentrum wie der VPS.
2. rclone-Remote einrichten:
   ```sh
   sudo -u reiterhof env RCLONE_CONFIG=/var/lib/reiterhof/rclone/rclone.conf rclone config
   ```
3. In `/etc/reiterhof/backup.env` `REITERHOF_RCLONE_REMOTE=<remote>:<pfad>` eintragen.
   Optional `REITERHOF_BACKUP_PING_URL` (z. B. healthchecks.io): meldet, wenn ein Backup
   ausbleibt.
4. Sofort testen: `sudo systemctl start reiterhof-backup.service && journalctl -u reiterhof-backup -n 30`.
   Ein fehlgeschlagenes Backup beendet die Unit mit Fehler (`systemctl --failed`).
5. Das Offsite-Ziel **muss** ein rclone-`crypt`-Remote sein (Verschlüsselung auf dem Server, bevor die
   Daten ihn verlassen): Die Stände enthalten personenbezogene Daten, und der Datenschutztext verspricht die
   Verschlüsselung. `backup.sh` prüft den Typ des Remotes und bricht die Offsite-Kopie sonst mit Fehler ab
   (`REITERHOF_ALLOW_PLAIN_OFFSITE=1` in `backup.env` schaltet die Prüfung ab, dann muss der Datenschutztext
   angepasst werden). Das Passwort des crypt-Remotes getrennt vom Server aufbewahren.

**Vorerst ohne Offsite-Kopie** (Stand 30.09.2026): `REITERHOF_SKIP_OFFSITE=1` in `/etc/reiterhof/backup.env`
eintragen, sonst meldet der Timer jede Nacht einen Fehler und der Ping bleibt aus. Alle Stände liegen dann auf
demselben Server; fällt er aus, sind Daten und Backups weg. Sobald ein Offsite-Ziel eingerichtet wird, die Schritte
1 bis 5 oben durchführen, den Datenschutztext (Abschnitte 5 bis 7) und `docs/legal/avv-checkliste.md` ergänzen.

## 7. Restore testen

Ein Backup gilt erst als vorhanden, wenn ein Restore geklappt hat. Anleitung und Testskript:
[`restore.md`](restore.md). Den Test einmal nach der Einrichtung und danach vierteljährlich
wiederholen.

## 8. Monitoring

Externen Uptime-Check einrichten, der von außerhalb des Servers prüft
(z. B. UptimeRobot, Better Stack oder Hetrix, alle kostenlos):

- URL: `https://<domain>/healthz`, Intervall 5 Minuten, erwartet HTTP 200
- Benachrichtigung per E-Mail/Push an dich
- Zusätzlich: SSL-Ablauf-Überwachung des Zertifikats, falls der Dienst das anbietet

Der Check erfasst Ausfälle von Caddy, API und Server. Für ausbleibende Backups siehe
`REITERHOF_BACKUP_PING_URL` oben.

## 9. Datenschutz: AVV mit Netcup

Netcup verarbeitet als Hosting-Anbieter personenbezogene Daten in deinem Auftrag. Nach
DSGVO (Art. 28) brauchst du dafür einen **Auftragsverarbeitungsvertrag (AVV/DPA)**:

1. Im Netcup Kundenkontrollpanel (CCP) den AVV abschließen (die genaue Stelle kann sich ändern; im CCP nach „Auftragsverarbeitung“ / „AVV“ suchen).
2. Bestätigten AVV als PDF ablegen.
3. Den Speicherort (Rechenzentrum Deutschland) und die Backup-Anbieter im Verzeichnis von
   Verarbeitungstätigkeiten festhalten; für den Backup-Speicher ebenfalls einen AVV abschließen.

Das ersetzt keine Rechtsberatung. Die Caddy-Konfiguration schreibt bewusst kein Access-Log
mit IP-Adressen.

## Betrieb: stallfunk-admin

Für alles, was die App (noch) nicht kann, gibt es das Kommandozeilenwerkzeug `stallfunk-admin`
statt SQL von Hand: den ersten Stall und Admin anlegen, Einladungscodes erzeugen, Nutzer
befördern, verschieben oder löschen, Pferde übertragen, Sitzungen beenden, Test-Mail senden,
Wetter neu laden, Migrationen prüfen. Es benutzt denselben Code wie die API (z. B. beim Löschen
von Konten) und liest `/etc/reiterhof/api.env`.

**Aufruf** (auf dem Server als `admin`; sudo passiert im Wrapper, `sudo stallfunk-admin ...` geht auch):

```sh
stallfunk-admin help
stallfunk-admin user list
```

Der Wrapper `/usr/local/bin/stallfunk-admin` startet `/opt/reiterhof/stallfunk-admin` als Benutzer
`reiterhof`: Nur dieser Benutzer (und root) darf `api.env` lesen, und er hat keine Rechte über die
Datenbank der API hinaus. Einstellungen kommen deshalb nur aus der Datei, nicht aus der Shell.
`--json` gibt Listen als JSON aus, `--yes` überspringt die Rückfrage bei `user delete`,
`user demote`, `user move` und `horse transfer`.

**Häufige Aufgaben**

```sh
# Erster Stall (Koordinaten für den Wetterabruf) und erster Admin
stallfunk-admin stable create --name "Stallgasse B" --farm "Hof Ahlers" --city Dorsten --lat 51.66 --lng 6.96
stallfunk-admin user create --email du@example.org --name "Dein Name" --admin
stallfunk-admin stable list

# Automatisches Abdecken (Stallpersonal nimmt die Decken ab, ohne die App zu nutzen):
# ab HH:MM (04:00 bis 14:59, Stallzeit) setzt der Job "Abgedeckt" für Pferde, die noch eingedeckt sind.
# Tage: mon-fri (Standard), all oder Liste wie mon,tue,sat (auch mo,di,mi,do,fr,sa,so); "off" schaltet aus.
stallfunk-admin stable update <stall-id> --auto-uncover 12:30 --auto-uncover-days mon-fri
stallfunk-admin stable update <stall-id> --auto-uncover off

# Einladungscode für neue Mitglieder (Standard: 7 Tage, 10 Einlösungen)
stallfunk-admin invite create
stallfunk-admin invite create --days 14 --max-uses 3
stallfunk-admin invite list

# Rechte und Zuordnung
stallfunk-admin user promote anna@example.org
stallfunk-admin user demote anna@example.org      # verweigert für den letzten Admin eines Stalls
stallfunk-admin user move anna@example.org --stable <stall-id>
stallfunk-admin horse transfer Luna --to anna@example.org
stallfunk-admin sessions revoke anna@example.org  # erzwingt neue Anmeldung

# Konto löschen (wie in der App: anonymisiert; blockiert, solange Pferde gehören oder letzter Admin)
stallfunk-admin user delete anna@example.org

# Mail-Versand prüfen (SMTP-Einstellungen aus api.env)
stallfunk-admin mail test --to du@example.org

# Wetter jetzt laden, Migrationen ansehen
stallfunk-admin weather refresh
stallfunk-admin migrate status
```

Gibt es genau einen Stall, gilt er als Standard für `--stable`; bei mehreren muss `--stable`
(ID oder Name) angegeben werden. Der neue Admin meldet sich in der App mit einem Magic Link an die
angegebene Adresse an (SMTP muss eingerichtet sein, siehe `docs/auth-setup.md`).

**Einmalig auf dem bestehenden Server nachrüsten**

Der Wrapper kommt mit `provision.sh`; das Binary liefert der nächste Deploy (Workflow *Deploy*
ausführen). Auf einem bereits eingerichteten Server reicht es, nur den Wrapper zu installieren
(vom eigenen Rechner aus, im Repository):

```sh
scp deploy/stallfunk-admin.sh admin@<server-ip>:/tmp/stallfunk-admin.sh
ssh admin@<server-ip> 'sudo install -m 0755 -o root -g root /tmp/stallfunk-admin.sh /usr/local/bin/stallfunk-admin && rm /tmp/stallfunk-admin.sh'
```

Alternativ das Repository auf dem Server aktualisieren und `provision.sh` erneut ausführen. Das
Datenbankpasswort liest das Skript bei einem erneuten Lauf aus `/etc/reiterhof/api.env`, es muss
nicht angegeben werden (ein abweichendes `REITERHOF_DB_PASSWORD` bricht ab). Die SSH-Schlüssel
werden aus den bestehenden Benutzern übernommen. `REITERHOF_WEB_DOMAIN` (Host der Web-App) ist
Pflicht; ohne sie bricht das Skript ab:

```sh
sudo bash -c 'cd /root/stallfunk && git pull --ff-only && cd deploy &&
  REITERHOF_DOMAIN=api.stallfunk.de \
  REITERHOF_WEB_DOMAIN=stallfunk.de \
  REITERHOF_ADMIN_SSH_PUBKEY="$(head -n1 /home/admin/.ssh/authorized_keys)" \
  REITERHOF_DEPLOY_SSH_PUBKEY="$(head -n1 /home/deploy/.ssh/authorized_keys)" \
  ./provision.sh'
```

## Caddyfile aktualisieren

Die Caddyfile kommt nur über `provision.sh` auf den Server; der Deploy-Workflow fasst sie nicht an.
Ändert sich nur die Caddyfile (z. B. die Security-Header), reicht es, sie einzeln einzuspielen.
Das Repository liegt auf dem Server unter `/root/stallfunk`; als `admin`:

```sh
sudo bash -c 'cd /root/stallfunk && git pull --ff-only &&
  set -a && . /etc/reiterhof/caddy.env && set +a &&
  caddy validate --config deploy/Caddyfile --adapter caddyfile &&
  install -m 0644 -o root -g root deploy/Caddyfile /etc/caddy/Caddyfile &&
  systemctl reload caddy'
```

`caddy validate` prüft die Datei mit den Domains aus `/etc/reiterhof/caddy.env`, bevor sie
installiert wird; `systemctl reload caddy` übernimmt sie ohne Unterbrechung. Kontrolle:

```sh
curl -sI https://stallfunk.de/auth/verify?token=aaaaaaaaaaaaaaaaaaaaaaaa | grep -i content-security-policy
# style-src 'sha256-…' und img-src data: kommen von der API, nicht das Caddy-Default "default-src 'none'"
curl -sI https://stallfunk.de/api/v1/me | grep -i content-security-policy   # Caddy-Default: default-src 'none'
```

Alternativ `provision.sh` erneut ausführen (Befehl unter [Betrieb](#betrieb-stallfunk-admin)); das
installiert die Caddyfile ebenfalls. Ohne Repository auf dem Server tut es auch `scp deploy/Caddyfile`
nach `/tmp` und dieselben Schritte ab `caddy validate` mit `/tmp/Caddyfile`.

## Wartung

- **Updates**: Sicherheitsupdates laufen automatisch (`unattended-upgrades`). Größere
  Updates (PostgreSQL-Major, Caddy) manuell und mit vorherigem Backup.
- **Rollback**: `ssh deploy@host` und `/opt/reiterhof/api.prev` nach `api` kopieren, dann
  `sudo systemctl restart reiterhof-api`. Web-App: Symlink umschalten, siehe [Web-App (PWA)](#web-app-pwa).
- **Logs**: `journalctl -u reiterhof-api`, `journalctl -u caddy`, `journalctl -u reiterhof-backup`.
