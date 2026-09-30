# Verzeichnis von Verarbeitungstätigkeiten, TOM und Abläufe

Für den Betreiber, nicht für die App. Verzeichnis nach Art. 30 Abs. 1 DSGVO, Beschreibung der technischen und organisatorischen Maßnahmen (Art. 32) und die Abläufe für Anfragen und Datenpannen. Stand: 30.09.2026. Bei jeder Änderung an Datenflüssen zusammen mit `datenschutz.md` anpassen.

## A. Allgemeine Angaben

- **Verantwortlicher:** Jan Krüger (Privatperson), jankrueger1999@gmail.com. Betreibt die App als Einstaller für die Einstaller seines Stalls, nicht gewerblich, nur auf Einladung.
- **Stallbetrieb:** betreibt die App nicht, entscheidet nicht über Zwecke und Mittel, hat keinen Zugang zu Server oder Admin-Werkzeug. Keine gemeinsame Verantwortlichkeit (Art. 26), solange das so bleibt (siehe Checkliste, Teil 4).
- **Datenschutzbeauftragter:** nicht benannt; nicht erforderlich (§ 38 BDSG: weniger als 20 Personen, keine Art. 35-Pflicht, siehe Abschnitt D).
- **Auftragsverarbeiter:** Netcup GmbH (Hosting, AVV vom 30.09.2026), Resend / Plus Five Five, Inc. (E-Mail, DPA mit EU-Standardvertragsklauseln, DPF), Mistral AI SAS, Paris (Sprachmodell für KI-Vorschläge im Wochenplan, nur mit Einwilligung des Besitzers; Training mit API-Anfragen im Konto ausgeschaltet; DPA siehe Checkliste, Teil 1).
- **Eigene Verantwortliche bzw. Übermittlungsempfänger:** Apple (APNs, Sign in with Apple, Apple Maps; Drittland USA über Standardvertragsklauseln), Google (FCM, Google Sign-In, Google Maps; Google LLC nach dem EU-US Data Privacy Framework zertifiziert), Push-Dienste der Browser (Apple, Google, Mozilla; Inhalt Ende-zu-Ende verschlüsselt), OpenFreeMap (Kartenkacheln der Web-App).
- **Betroffene allgemein:** Einstaller und Reitbeteiligungen (Nutzer, auch Minderjährige), Erziehungsberechtigte minderjähriger Nutzer, von Nutzern eingetragene Kontaktpersonen (Notfallkarte, Tierarzt).
- **Keine** besonderen Kategorien (Art. 9), kein Profiling, keine automatisierten Entscheidungen, keine Werbung, keine Analyse-Dienste. Die KI-Vorschläge im Wochenplan (B9) betreffen Pferde, werden von festen Regeln geprüft und nur übernommen, wenn eine Person das bestätigt.

## B. Verarbeitungstätigkeiten

Rechtsgrundlagen und Einzelheiten stehen im Datenschutztext (Abschnittsnummern in Klammern). Speicherort aller Daten: Server bei Netcup (Deutschland), Backups auf demselben Server.

### B1. Konto und Anmeldung (3.1)

- **Zweck:** Zugang zur App, Zuordnung zum Stall, Rechte.
- **Betroffene:** Nutzer.
- **Daten:** Name, E-Mail-Adresse, optional Telefonnummer, Avatar-Farbe, Stall, Admin-Kennzeichen, Registrierungszeitpunkt; Hashes von Anmelde-Links und -Codes; Sitzungs-Hash, User-Agent, Zeitpunkte; bei Google/Apple-Anmeldung die Kennung beim Anbieter (`sub`).
- **Empfänger:** Mitglieder des Stalls (Name, Farbe, Telefonnummer auf der Notfallkarte), Resend (Anmelde-Mail), Google/Apple (bei deren Anmeldung).
- **Löschung:** Konto bis zur Löschung durch die Person; Anmeldeversuche nach 15 Minuten ungültig, nach spätestens einem Tag gelöscht; Sitzungen 90 Tage nach letzter Nutzung.

### B2. Anwesenheit im Stall und Geofence (3.2, 3.3)

- **Zweck:** Absprachen im Stall (wer ist da).
- **Betroffene:** Nutzer mit Einwilligung.
- **Daten:** Ankunft, Abgang, Quelle (Hand/Geofence), Sichtbarkeit. Der Standort selbst verlässt das Gerät nicht.
- **Empfänger:** Mitglieder des Stalls nach der gewählten Sichtbarkeit.
- **Löschung:** abgeschlossene Besuche nach 12 Monaten (Job `privacy-retention`).

### B3. Training, GPS-Strecken, Karten (3.4)

- **Zweck:** Trainingsdokumentation für Reiter und Pferd.
- **Betroffene:** Nutzer mit Einwilligung.
- **Daten:** Trainingseinheit (Dauer, Strecke, Gangarten), Positionspunkte, Sensormerkmale je Zeitabschnitt.
- **Empfänger:** Besitzer, Reitbeteiligungen des Pferdes, Admins; beim Anzeigen der Karte sieht der Kartendienst (Apple, Google, OpenFreeMap) IP-Adresse und Kartenausschnitt, nur nach Einwilligung „Karten anzeigen“.
- **Löschung:** Positionspunkte und Sensormerkmale nach 12 Monaten; die Einheit selbst bleibt als Trainingsverlauf des Pferdes.

### B4. Pferdeakte, Notfallkarte, Dokumente, Fotos (3.5 bis 3.7)

- **Zweck:** gemeinsame Versorgung der Pferde, Hilfe im Notfall.
- **Betroffene:** Nutzer (Besitzer, Melder, Reitbeteiligungen), Kontaktpersonen auf der Notfallkarte, ggf. Personen auf Fotos.
- **Daten:** Angaben zu Pferden mit Bezug zu Besitzer und Eintragendem, Telefonnummern und Namen auf der Notfallkarte, Fotos (Metadaten entfernt), Dokumente (PDF unverändert).
- **Empfänger:** Mitglieder des Stalls (Notfallkarte, Besitz, Reitbeteiligungen, Fotos von Auffälligkeiten und Decken), Dokumente nur Besitzer, Reitbeteiligungen, Admins.
- **Löschung:** solange gebraucht, Löschung durch Besitzer oder Admin; bei Kontolöschung anonymisiert („Gelöschtes Mitglied“), Fotos der eigenen Meldungen gelöscht.

### B5. Anfragen, Decken, Wochen- und Trainingsplan (3.8)

- **Zweck:** Organisation der Versorgung im Stall.
- **Betroffene:** Nutzer.
- **Daten:** wer etwas angefragt, zugesagt, eingetragen oder geändert hat; Ersteller von Einladungscodes.
- **Empfänger:** Mitglieder des Stalls.
- **Löschung:** solange gebraucht; Einladungscodes 30 Tage nach Ablauf.

### B6. Benachrichtigungen und Erinnerungen (3.9)

- **Zweck:** Erinnerungen an Termine, Anfragen, Decken.
- **Betroffene:** Nutzer mit Einwilligung.
- **Daten:** Gerätetoken mit Plattform bzw. Browser-Push-Adresse mit Schlüsseln und User-Agent; Nachrichtentext (enthält z. B. Pferdename und Name des Melders); versendete Erinnerungen.
- **Empfänger:** Apple (APNs), Google (FCM); in der Web-App der Push-Dienst des Browsers (Inhalt verschlüsselt).
- **Löschung:** Tokens beim Widerruf der Einwilligung und bei Kontolöschung; versendete Erinnerungen nach 12 Monaten.

### B7. Einwilligungen, Altersangabe, Zustimmung der Eltern (4)

- **Zweck:** Nachweis der Einwilligungen (Art. 7 Abs. 1) und der Zustimmung der Eltern (Art. 8).
- **Betroffene:** Nutzer, Erziehungsberechtigte.
- **Daten:** Art, Textversion und Zeitpunkte der Einwilligungen; Angabe „16 oder älter“ mit Zeitpunkt; E-Mail-Adresse des Elternteils, Hash des Links, Zeitpunkt der Zustimmung.
- **Empfänger:** Resend (Mail an das Elternteil), das Elternteil (Name und E-Mail-Adresse des Kindes).
- **Löschung:** solange das Konto besteht; Links an Eltern nach 7 Tagen.

### B9. KI-Vorschläge im Wochenplan (3.12)

- **Zweck:** Vorschlag für die offenen Trainingstage einer Woche.
- **Betroffene:** Besitzer mit Einwilligung (`ai_training`); Reitbeteiligungen und andere Eintragende nur mittelbar über die Einheiten ihres Pferdes, ohne Namen und ohne Datum.
- **Daten:** Disziplin, Status, Rhythmus, freigegebene Aktivitäten, Einheiten der letzten 14 Tage (Tage zurück, Aktivität, Minuten, Belastung, Galopp-Anteil, Gefühl), Turniere in Tagen, Reha-Vorgaben je Tag, Wetter und Boden von heute, belegte Tage der Woche. Keine Namen, Kennungen, Freitexte, Kalenderdaten oder Standortdaten (`backend/internal/training/weekplan`, Funktion `Prompt`, mit Test).
- **Empfänger:** Mistral AI SAS (Auftragsverarbeiter, EU).
- **Rechtsgrundlage:** Einwilligung des Besitzers (Art. 6 Abs. 1 lit. a), für die Einheiten anderer Personen berechtigtes Interesse (lit. f).
- **Löschung:** Anfrage und Antwort werden auf dem Server nicht gespeichert; bei Mistral laut Anbieter bis zu 30 Tage (Missbrauchskontrolle). Übernommene Tage liegen in `week_slots` wie andere Einträge (B5).

### B8. Betrieb: Protokolle, Missbrauchsschutz, Backups (5)

- **Zweck:** sicherer Betrieb, Wiederherstellung.
- **Betroffene:** alle.
- **Daten:** keine Zugriffsprotokolle; Fehlerprotokolle von API und Caddy im Systemjournal (können bei Verbindungsfehlern IP-Adressen enthalten); langsame Datenbankabfragen im PostgreSQL-Log (ohne Parameterwerte); IP-Adressen im Arbeitsspeicher für Rate-Limits (höchstens 24 Stunden); vollständige Backups.
- **Empfänger:** keine.
- **Löschung:** Backups 14 tägliche und 8 wöchentliche Stände (höchstens 8 Wochen); Systemjournal nach vier Wochen (`MaxRetentionSec=4week`); PostgreSQL-Logs nach logrotate der Distribution (enthalten keine Werte aus den Abfragen).

## C. Technische und organisatorische Maßnahmen (Art. 32)

Quelle: `deploy/` (Runbook `deploy/README.md`), Anwendungscode.

- **Zutritt:** Rechenzentrum der Netcup GmbH, Maßnahmen laut AVV (Anhang 1).
- **Zugang zum Server:** SSH nur mit Schlüssel (`PasswordAuthentication no`, `PermitRootLogin no`, `AllowUsers admin deploy`, `MaxAuthTries 3`), Passwörter der Login-Benutzer gesperrt; Firewall `ufw` lässt nur 22 (rate-limitiert), 80 und 443 zu.
- **Zugriff:** Nur der Benutzer `admin` (der Betreiber) hat volle Rechte. Der Deploy-Schlüssel (GitHub Actions) darf nur Dateien in `/opt/reiterhof` ablegen und den Dienst neu starten. Die API läuft als Systembenutzer `reiterhof` ohne Shell, mit systemd-Sandbox (`ProtectSystem=strict`, `NoNewPrivileges`, leere Capabilities, Systemaufruf-Filter). Geheimnisse (`/etc/reiterhof/*.env`, APNs- und FCM-Schlüssel) sind nur für root und die Gruppe `reiterhof` lesbar. `stallfunk-admin` läuft nur per sudo.
- **In der App:** Rollen (Mitglied, Reitbeteiligung, Besitzer, Admin) werden auf dem Server geprüft; jede Abfrage ist auf den Stall beschränkt; Dateien werden nur mit der Sichtbarkeit des zugehörigen Eintrags ausgeliefert; Einwilligungen werden auf dem Server durchgesetzt.
- **Weitergabe und Transport:** TLS über Caddy (Let's Encrypt, HSTS), strenge Sicherheits-Header und CSP; API und Datenbank lauschen nur auf `localhost`; Mails per STARTTLS; Web-Push Ende-zu-Ende verschlüsselt; Anfragen an Mistral per TLS, nur mit Einwilligung des Besitzers und ohne Namen oder Freitext; der API-Schlüssel (`REITERHOF_MISTRAL_API_KEY`) liegt in `/etc/reiterhof/api.env`; Labs-Modelle lehnt der Server ab.
- **Speicherung:** Sitzungs-Schlüssel, Anmelde-Links und Links an Eltern nur als SHA-256-Hash, Anmelde-Codes als HMAC; Datenbank-Passwort mit SCRAM-SHA-256. Keine eigene Festplattenverschlüsselung auf dem VPS, lokale Backups unverschlüsselt (nur Dateirechte 0700/0750).
- **Datenminimierung:** keine Zugriffsprotokolle mit IP-Adressen; Systemjournal höchstens vier Wochen, Datenbank-Protokolle ohne Parameterwerte; Metadaten aus Fotos entfernt; Geofence ohne Standortübertragung; Löschfristen automatisch (Job `privacy-retention`, täglich 03:30).
- **Missbrauchsschutz:** Rate-Limits für Anmeldung, Code-Prüfung, Beitritt und Mails an Eltern; die Anmeldung verrät nicht, ob eine Adresse registriert ist.
- **Verfügbarkeit:** tägliches Backup 03:00 (Datenbank und Dateien), 14 tägliche und 8 wöchentliche Stände, geprüft mit `pg_restore --list`; Restore-Anleitung `deploy/restore.md` (RPO 24 h), Restore-Test vierteljährlich (`deploy/restore-test.sh`). **Kein Offsite-Backup:** fällt der Server aus, sind Daten und Backups weg.
- **Aktualität:** unattended-upgrades mit automatischem Neustart 04:30; größere Updates (PostgreSQL, Caddy) von Hand nach einem Backup.
- **Überprüfung:** diese Liste bei Änderungen an `deploy/` prüfen; Restore-Test vierteljährlich.

## D. Datenschutz-Folgenabschätzung (Schwellwertanalyse, Art. 35)

Kriterien der Artikel-29-Gruppe (WP 248) und der Muss-Liste der Datenschutzkonferenz:

- **Standortdaten:** ja, aber nur freiwillig (Einwilligung, jederzeit abschaltbar); Geofence bleibt auf dem Gerät, GPS-Strecken nur bei selbst gestarteter Aufzeichnung.
- **Schutzbedürftige Betroffene:** ja, Minderjährige können Nutzer sein (mit Zustimmung der Eltern).
- **Umfang:** nein, ein Stall, einige Dutzend Personen.
- **Neue Technologien:** KI (Sprachmodell) für Trainingsvorschläge, aber sie bewertet Pferde und keine Personen, bekommt keine Namen, entscheidet nichts (feste Regeln prüfen, eine Person übernimmt) und läuft nur mit Einwilligung. Kein Fall der Muss-Liste (die dort genannten KI-Fälle betreffen die Bewertung oder Steuerung von Personen).
- **Systematische Überwachung, Bewertung, Scoring, Zusammenführung von Datenbeständen, Ausschluss von Rechten:** nein.

Ergebnis: Zwei Kriterien sind berührt, aber der Umfang ist klein und die Verarbeitung von Standortdaten freiwillig und für die Betroffenen steuerbar; kein Fall der Muss-Liste. Eine vollständige Folgenabschätzung ist nach dieser Einschätzung nicht erforderlich. Neu prüfen, wenn die App für mehrere Ställe oder deutlich mehr Personen betrieben wird oder Standortdaten ohne eigenes Zutun erhoben werden.

## E. Abläufe

### E1. Auskunft, Export, Berichtigung, Löschung (Art. 15 bis 20)

1. Anfrage per E-Mail an jankrueger1999@gmail.com. Identität prüfen: Die Anfrage muss von der im Konto hinterlegten Adresse kommen, sonst Rückfrage.
2. **Export und Löschung:** auf die App verweisen (Einstellungen, Datenschutz, „Daten exportieren“ bzw. „Konto löschen“). Kann die Person die App nicht nutzen: Löschung mit `stallfunk-admin user delete <E-Mail>`; einen Export per Admin-Werkzeug gibt es nicht, dann die Daten per SQL zusammenstellen (Tabellen wie in `backend/internal/privacy/export.go`).
3. Besitzt die Person noch Pferde oder ist sie letzter Admin, verweigert das Werkzeug die Löschung: Pferde mit `stallfunk-admin horse transfer` übergeben bzw. mit `user promote` einen neuen Admin bestimmen, dann löschen.
4. Frist: ein Monat (Art. 12 Abs. 3). Antwort und Datum kurz notieren (z. B. im Postfach-Ordner „Datenschutz“).
5. Gelöschte Daten verschwinden aus den Backups nach spätestens 8 Wochen. Nach einem Restore die Löschungen seit dem Backup wiederholen (`deploy/restore.md`).

### E2. Widerruf der Zustimmung der Eltern

1. Das Elternteil schreibt an die E-Mail-Adresse aus Abschnitt 1 des Datenschutztexts.
2. Prüfen, ob die Absenderadresse die gespeicherte Adresse des Elternteils ist (`users.parent_email`); sonst Rückfrage.
3. Konto löschen (`stallfunk-admin user delete <E-Mail des Kindes>`, ggf. vorher Pferde übergeben), Elternteil und Kind kurz informieren.
4. Das Postfach regelmäßig lesen; eine Weiterleitung aufs Handy genügt.

### E3. Datenpanne (Art. 33, 34)

1. **Erkennen:** z. B. Server kompromittiert, Backup oder Schlüssel abhanden, Daten für falsche Personen sichtbar, Mail an falschen Empfänger.
2. **Eindämmen:** Zugang sperren (SSH-Schlüssel tauschen, `stallfunk-admin sessions revoke`, Geheimnisse in `/etc/reiterhof` erneuern: Login-Code-Schlüssel, APNs/FCM, SMTP), Fehler beheben.
3. **Bewerten:** Welche Daten, wie viele Personen, welches Risiko? Standortdaten, Telefonnummern und Daten Minderjähriger sprechen für ein Risiko.
4. **Melden:** Ist ein Risiko für die Betroffenen nicht unwahrscheinlich, binnen 72 Stunden nach Bekanntwerden an die Aufsichtsbehörde des eigenen Bundeslandes melden (Online-Formular der Behörde). Bei hohem Risiko auch die Betroffenen benachrichtigen (Art. 34), z. B. per Mail an alle Mitglieder.
5. **Dokumentieren:** jede Panne, auch ohne Meldung, mit Datum, Sachverhalt, Folgen und Maßnahmen (Art. 33 Abs. 5).
