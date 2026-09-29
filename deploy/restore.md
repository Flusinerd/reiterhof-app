# Restore aus dem Backup

Backups liegen an zwei Orten:

- lokal auf dem Server: `/var/backups/reiterhof/daily` (14 Tage) und `/weekly` (8 Wochen)
- offsite: unter `REITERHOF_RCLONE_REMOTE` (`daily/` und `weekly/`)

Pro Tag gibt es zwei Dateien:

- `reiterhof-YYYY-MM-DD.dump` — `pg_dump -Fc` der Datenbank
- `reiterhof-YYYY-MM-DD-uploads.tar.gz` — Verzeichnis `uploads/` (Fotos)

## A. Restore testen (ohne Auswirkungen auf Produktion)

Das Skript `restore-test.sh` (installiert als `/usr/local/sbin/reiterhof-restore-test`)
spielt einen Dump in eine Wegwerf-Datenbank `reiterhof_restoretest` ein, zählt die
Zeilen aller Tabellen, entpackt das Upload-Archiv in ein temporäres Verzeichnis und räumt
alles wieder auf:

```sh
D=$(date +%F)
sudo reiterhof-restore-test \
  /var/backups/reiterhof/daily/reiterhof-$D.dump \
  /var/backups/reiterhof/daily/reiterhof-$D-uploads.tar.gz
```

Erwartet: `Restore test passed`, plausible Zeilenzahlen (z. B. Pferde, Benutzer) und eine
Dateizahl im Upload-Archiv, die zum Bestand passt. Exit-Code ungleich 0 heißt: das Backup ist
nicht brauchbar, sofort untersuchen (`journalctl -u reiterhof-backup`).

**Offsite-Kopie testen**: gleicher Ablauf mit Dateien vom Offsite-Speicher:

```sh
sudo -u reiterhof env RCLONE_CONFIG=/var/lib/reiterhof/rclone/rclone.conf \
  rclone copy "$REMOTE/daily/reiterhof-$D.dump" /tmp/restore-check/
sudo reiterhof-restore-test /tmp/restore-check/reiterhof-$D.dump
sudo rm -rf /tmp/restore-check
```

(`$REMOTE` ist der Wert von `REITERHOF_RCLONE_REMOTE`; hat das Ziel eine `crypt`-Schicht,
sind die Dateien nach dem `rclone copy` bereits entschlüsselt.)

Test einmal nach der Einrichtung und danach vierteljährlich durchführen.

## B. Ernstfall: Produktion wiederherstellen

Die folgenden Befehle als root ausführen (`sudo -i`), weil die Backup-Dateien nur für
den Benutzer `reiterhof` und root lesbar sind.

### Fall 1: Datenbank kaputt oder falsche Daten, Server intakt

```sh
systemctl stop reiterhof-api
# Sicherheitskopie des aktuellen Stands
runuser -u postgres -- pg_dump -Fc reiterhof > /root/reiterhof-before-restore.dump
# Datenbank neu anlegen und einspielen (der Dump wird über stdin gelesen)
runuser -u postgres -- dropdb reiterhof
runuser -u postgres -- createdb --owner reiterhof reiterhof
runuser -u postgres -- pg_restore --exit-on-error --no-owner --role reiterhof \
  --dbname reiterhof < /var/backups/reiterhof/daily/reiterhof-YYYY-MM-DD.dump
systemctl start reiterhof-api
curl -fsS http://127.0.0.1:8080/healthz
```

`--no-owner --role reiterhof` sorgt dafür, dass alle Objekte der Anwendungsrolle gehören.

### Fall 2: Uploads wiederherstellen

```sh
systemctl stop reiterhof-api
tar --extract --gzip --file /var/backups/reiterhof/daily/reiterhof-YYYY-MM-DD-uploads.tar.gz \
  --directory /var/lib/reiterhof --no-same-owner
chown -R reiterhof:reiterhof /var/lib/reiterhof/uploads
systemctl start reiterhof-api
```

### Fall 3: Server verloren

1. Neuen VPS bestellen, DNS-Records auf die neue IP umstellen (TTL vorher niedrig halten).
2. `deploy/provision.sh` ausführen (siehe `README.md`, Schritte 1–4).
3. Dump und Upload-Archiv vom Offsite-Speicher holen (`rclone copy`, siehe oben) und mit
   Fall 1 und Fall 2 einspielen.
4. Deployment über GitHub Actions auslösen (neue Werte für `DEPLOY_HOST` und `DEPLOY_KNOWN_HOSTS`).
5. `https://<domain>/healthz` prüfen.

Zielwerte: Datenverlust höchstens 24 Stunden (RPO), Wiederherstellung innerhalb eines
halben Tages (RTO), sofern der Offsite-Speicher erreichbar ist.
