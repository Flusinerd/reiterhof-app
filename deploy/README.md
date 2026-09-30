# Deployment auf einen Netcup-VPS

Runbook für den Betrieb der Stallfunk-API auf **einem** Debian-/Ubuntu-Server
(Netcup-VPS mit 4 GB RAM). Alles läuft auf dem Server:

```
Internet -> Caddy (80/443, automatisches TLS) -> reiterhof-api (127.0.0.1:8080) -> PostgreSQL 16 (localhost)
```

| Datei | Zweck |
| --- | --- |
| `provision.sh` | Einmalige, wiederholbare Server-Einrichtung (als root) |
| `Caddyfile` | Reverse Proxy, TLS, Security-Header, Upload-Limit 25 MB |
| `postgresql.conf.d/reiterhof.conf` | PostgreSQL-Tuning für 4 GB RAM |
| `systemd/reiterhof-api.service` | API als gehärteter systemd-Dienst |
| `api.env.example` | Vorlage für `/etc/reiterhof/api.env` |
| `backup.sh`, `systemd/reiterhof-backup.*` | Tägliches Backup + Offsite-Kopie |
| `restore-test.sh`, `restore.md` | Restore-Anleitung und Restore-Test |
| `remote-swap.sh` | Wird vom Deploy-Workflow auf dem Server ausgeführt |
| `stallfunk-admin.sh` | Wrapper `/usr/local/bin/stallfunk-admin` für das Betriebswerkzeug (siehe [Betrieb](#betrieb-stallfunk-admin)) |

## 1. Server bestellen

1. Bei Netcup einen VPS mit 4 GB RAM bestellen (z. B. VPS Lite 1: 2 vCore, 4 GB RAM, 80 GB SSD) (Rechenzentrum in Deutschland,
   damit die Daten in der EU bleiben).
2. Image: **Debian 13 (Minimal)** empfohlen; Debian 12 und Ubuntu 24.04 funktionieren ebenfalls. Beim Anlegen deinen SSH-Public-Key
   hinterlegen (Server Control Panel, SCP). Beide Distributionen werden unterstützt.
3. Die IPv4-/IPv6-Adresse des Servers notieren.

## 2. DNS

Bei deinem DNS-Anbieter für die API-Domain (z. B. `api.reiterhof.example`) anlegen:

- `A`-Record auf die IPv4-Adresse,
- `AAAA`-Record auf die IPv6-Adresse (optional, aber empfohlen).

Prüfen: `dig +short api.reiterhof.example`. Ohne korrekten DNS-Eintrag kann Caddy kein
Zertifikat bekommen. Ports 80 und 443 müssen erreichbar sein (die Firewall wird von
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
  `/var/lib/reiterhof/uploads`, `/var/backups/reiterhof`
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
externen Health-Check am Ende) und `DEPLOY_ENABLED=true` (automatisches Deployment, sobald die CI
für einen Push auf `main` grün ist; ausgerollt wird genau der getestete Commit. Ohne diese
Variable läuft der Workflow nur manuell).

Dann *Actions > Deploy > Run workflow*. Der Workflow testet, baut zwei statische
linux/amd64-Binaries (API und das Betriebswerkzeug `stallfunk-admin`), lädt sie per `scp` hoch,
tauscht `/opt/reiterhof/api` atomar aus, startet den Dienst neu und prüft `/healthz`. Fällt der
Health-Check durch, wird automatisch das vorherige Binary wiederhergestellt
(`/opt/reiterhof/api.prev`). Das Betriebswerkzeug wird erst nach erfolgreichem Health-Check als
`/opt/reiterhof/stallfunk-admin` installiert (ebenfalls atomar, ohne zusätzliche sudo-Rechte).

Kontrolle auf dem Server:

```sh
systemctl status reiterhof-api
journalctl -u reiterhof-api -f
curl -fsS https://api.reiterhof.example/healthz
```

Datenbank-Migrationen laufen beim Start der API automatisch.

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
5. Stände auf dem Offsite-Speicher sollten verschlüsselt sein (rclone `crypt`-Remote), weil
   sie personenbezogene Daten enthalten.

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

Alternativ das Repository aktualisieren (`git pull`) und `sudo ./provision.sh` erneut ausführen,
dann aber mit **denselben** Werten wie beim ersten Mal, insbesondere `REITERHOF_DB_PASSWORD`: Das
Skript setzt das Datenbankpasswort neu, `api.env` wird aber nicht überschrieben. Ein anderes
Passwort würde die API von der Datenbank aussperren.

## Wartung

- **Updates**: Sicherheitsupdates laufen automatisch (`unattended-upgrades`). Größere
  Updates (PostgreSQL-Major, Caddy) manuell und mit vorherigem Backup.
- **Rollback**: `ssh deploy@host` und `/opt/reiterhof/api.prev` nach `api` kopieren, dann
  `sudo systemctl restart reiterhof-api`.
- **Logs**: `journalctl -u reiterhof-api`, `journalctl -u caddy`, `journalctl -u reiterhof-backup`.
