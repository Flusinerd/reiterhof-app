# Checkliste: Verträge, Platzhalter und offene Fragen

**Entwurf – vor Veröffentlichung rechtlich prüfen lassen.** Diese Liste ist für den Betreiber (nicht für die App). Sie sagt, was vor dem Start unterschrieben oder ausgefüllt sein muss. Sie ersetzt keine Rechtsberatung.

## 1. Auftragsverarbeitungsverträge (AVV, Art. 28 DSGVO)

Ein AVV ist nötig, wenn ein Dienstleister in deinem Auftrag personenbezogene Daten verarbeitet.

- [ ] **Netcup GmbH (Hosting, VPS).** Im Netcup-Kundenkontrollpanel (CCP) den AVV abschließen, Bestätigung als PDF ablegen. Stelle im CCP nach „Auftragsverarbeitung“ / „AVV“ suchen; sie kann sich ändern. Siehe auch `deploy/README.md`, Abschnitt 9.
- [ ] **Offsite-Backup-Speicher** (Ziel des `rclone`-Kopierens, siehe `deploy/README.md`, Abschnitt 6). Eigener AVV mit diesem Anbieter, Standort EU, Remote als `crypt`-Remote verschlüsseln. Anbieter im Datenschutztext (`[Anbieter des Backup-Speichers]`) eintragen.
- [ ] **E-Mail-/SMTP-Anbieter** für die Anmelde-Links (`REITERHOF_SMTP_*`, siehe `docs/auth-setup.md`). AVV abschließen bzw. prüfen, ob er in den Nutzungsbedingungen enthalten ist. Anbieter im Datenschutztext eintragen.
- [ ] **Domain-/DNS-Anbieter:** nur nötig, wenn dort personenbezogene Daten anfallen (in der Regel nicht).

## 2. Expo, Google und Apple

Hier gibt es in der Regel keinen individuell ausgehandelten AVV. Es gelten die Bedingungen der Anbieter, die du mit dem Entwickler- bzw. Dienstkonto akzeptierst.

- [ ] **Expo (Push-Dienst `exp.host`, ggf. EAS-Build):** Nutzungsbedingungen, Datenschutzerklärung und ein angebotenes Data Processing Addendum (DPA) auf expo.dev prüfen und annehmen, soweit angeboten. Sitz in den USA: Drittlandübermittlung im Datenschutztext (Abschnitt 3.9) prüfen.
- [ ] **Apple (Apple Developer Program, APNs, Sign in with Apple):** Apple Developer Program License Agreement gilt. Datenschutzangaben im App Store Connect („App-Datenschutz“) ausfüllen.
- [ ] **Google (Firebase Cloud Messaging für Android-Push, Google Sign-In, Play Console):** Google-Bedingungen für Entwickler und Datenverarbeitung gelten. Die „Data safety“-Angaben in der Play Console ausfüllen.
- [ ] Prüfen, ob die drei Anbieter im Datenschutztext (Abschnitt 5) zutreffend beschrieben sind.

## 3. Platzhalter im Text ausfüllen

Quelle der Texte: `docs/legal/datenschutz.md` und `docs/legal/impressum.md`. Nach jeder Änderung `node scripts/gen-legal.mjs` ausführen (siehe `docs/domains/privacy.md`).

- [ ] Verantwortlicher: `[Name, Anschrift, E-Mail]` (Datenschutz, Abschnitt 1 und 8)
- [ ] Name des Stalls: `[Name des Stalls]`
- [ ] `[E-Mail-Anbieter für den Versand]` (Abschnitte 3.1, 5)
- [ ] `[Anbieter des Backup-Speichers, Standort EU]` (Abschnitte 4, 5)
- [ ] `[zuständige Aufsichtsbehörde, Anschrift, Website]` (Abschnitt 8)
- [ ] Hinweise `[Abschluss bestätigen]` und `[vor Veröffentlichung prüfen]` entfernen, sobald erledigt
- [ ] Impressum: Name, Anschrift, E-Mail, Telefon, Verbraucherstreitbeilegung
- [ ] Die Marke „Entwurf – vor Veröffentlichung rechtlich prüfen lassen“ aus beiden Texten entfernen, wenn die Prüfung abgeschlossen ist.
- [ ] Bei jeder inhaltlichen Änderung `TextVersion` in `backend/internal/privacy/consents.go` und die „Textversion“ in den Texten gemeinsam erhöhen (der Test `TestLegalTextsMatchTextVersion` schlägt sonst fehl).

## 4. Fragen für die rechtliche Prüfung

- [ ] Gilt die DSGVO hier, oder greift die Haushaltsausnahme (Art. 2 Abs. 2 lit. c)? Bei einer App für mehrere Einstaller, betrieben von einer Person, eher nicht.
- [ ] Impressumspflicht für ein privates, nicht geschäftsmäßiges Angebot.
- [ ] Rechtsgrundlagen je Datenkategorie (Einwilligung vs. Vertrag vs. berechtigtes Interesse), insbesondere Notfallkarte mit Telefonnummern für alle Mitglieder.
- [ ] Standortdaten (Geofence, GPS-Strecken): reicht die Einwilligung in der App, ist eine Datenschutz-Folgenabschätzung (Art. 35) nötig?
- [ ] Beschäftigtendaten- oder Vereinsbezug? Ist der Stallbetreiber (Hofbesitzer) Mitverantwortlicher?
- [ ] Umgang mit Fotos, auf denen Personen zu sehen sind.
- [ ] Drittlandübermittlung an Expo (USA): Angemessenheitsbeschluss oder Standardvertragsklauseln.
- [ ] Bewusste Entscheidung: Beim Löschen eines Kontos bleiben Einträge zu Pferden anonymisiert („Gelöschtes Mitglied“) erhalten; Besitzer müssen ihre Pferde vorher übergeben.

## 5. Weitere Pflichten, die nicht in der App liegen

- [ ] Verzeichnis von Verarbeitungstätigkeiten (Art. 30) anlegen (Vorlage: Kategorien aus dem Datenschutztext, Abschnitte 3 bis 6).
- [ ] Technisch-organisatorische Maßnahmen dokumentieren (Server-Härtung, siehe `deploy/`, Backups, Verschlüsselung, Zugriff nur für Admin-Benutzer).
- [ ] Ablauf für Auskunfts- und Löschanfragen festlegen. Die App erledigt Export und Löschung selbst; Anfragen per E-Mail kannst du mit den Endpunkten `GET /api/v1/me/export` und `POST /api/v1/me/delete` bzw. per SQL erledigen.
- [ ] Ablauf bei Datenpannen (Meldung binnen 72 Stunden an die Aufsichtsbehörde, Art. 33).
- [ ] Restore-Test der Backups (Vierteljahr), siehe `deploy/restore.md`; nach einem Restore die Löschungen seit dem Backup erneut ausführen.
- [ ] Datenschutz-Links in App Store Connect und Play Console eintragen (URL der Datenschutzerklärung; die App zeigt den Text auch selbst an).
