# Checkliste: Verträge, Platzhalter und offene Fragen

**Entwurf – vor Veröffentlichung rechtlich prüfen lassen.** Diese Liste ist für den Betreiber (nicht für die App). Sie sagt, was vor dem Start unterschrieben oder ausgefüllt sein muss. Sie ersetzt keine Rechtsberatung.

## 1. Auftragsverarbeitungsverträge (AVV, Art. 28 DSGVO)

Ein AVV ist nötig, wenn ein Dienstleister in deinem Auftrag personenbezogene Daten verarbeitet.

- [ ] **Netcup GmbH (Hosting, VPS).** Im Netcup-Kundenkontrollpanel (CCP) den AVV abschließen, Bestätigung als PDF ablegen. Stelle im CCP nach „Auftragsverarbeitung“ / „AVV“ suchen; sie kann sich ändern. Siehe auch `deploy/README.md`, Abschnitt 9.
- [ ] **Offsite-Backup-Speicher** (Ziel des `rclone`-Kopierens, siehe `deploy/README.md`, Abschnitt 6). Eigener AVV mit diesem Anbieter, Standort EU, Remote als `crypt`-Remote verschlüsseln. Anbieter im Datenschutztext (`[Anbieter des Backup-Speichers]`) eintragen.
- [ ] **Resend** (E-Mail-Versand der Anmelde-Links und der Mails an Eltern, `REITERHOF_SMTP_*`, siehe `docs/auth-setup.md`). Das DPA von Resend prüfen/akzeptieren und ablegen; Grundlage der US-Übermittlung (Data Privacy Framework oder Standardvertragsklauseln) prüfen. Im Datenschutztext bereits eingetragen.
- [ ] **Offsite-Backup verschlüsselt:** `backup.sh` kopiert nur auf ein rclone-`crypt`-Remote (sonst bricht es ab). Das Passwort des crypt-Remotes getrennt vom Server sichern, sonst ist das Backup wertlos.
- [ ] **Domain-/DNS-Anbieter:** nur nötig, wenn dort personenbezogene Daten anfallen (in der Regel nicht).

## 2. Expo, Google und Apple

Hier gibt es in der Regel keinen individuell ausgehandelten AVV. Es gelten die Bedingungen der Anbieter, die du mit dem Entwickler- bzw. Dienstkonto akzeptierst.

- [ ] **Expo (Push-Dienst `exp.host`, ggf. EAS-Build):** Nutzungsbedingungen, Datenschutzerklärung und ein angebotenes Data Processing Addendum (DPA) auf expo.dev prüfen und annehmen, soweit angeboten. Sitz in den USA: Drittlandübermittlung im Datenschutztext (Abschnitt 3.9) prüfen.
- [ ] **Apple (Apple Developer Program, APNs, Sign in with Apple):** Apple Developer Program License Agreement gilt. Datenschutzangaben im App Store Connect („App-Datenschutz“) ausfüllen.
- [ ] **Google (Firebase Cloud Messaging für Android-Push, Google Sign-In, Play Console, Google Maps SDK für die Ausritt-Karte):** Google-Bedingungen für Entwickler und Datenverarbeitung gelten; für Maps zusätzlich die Google Maps Platform Terms (API-Key, siehe `docs/domains/tracking.md`). Die „Data safety“-Angaben in der Play Console ausfüllen.
- [ ] **Kartendienste:** Apple Maps (MapKit, iOS) läuft über das Apple Developer Agreement; OpenFreeMap (Web-App) ist ein kostenloses Angebot ohne Vertrag und ohne Zusagen (Alternative: eigener Tile-Server, `EXPO_PUBLIC_MAP_STYLE_URL`). Karten werden erst nach der Einwilligung „Karten anzeigen“ geladen.
- [ ] Prüfen, ob die Anbieter im Datenschutztext (Abschnitt 6) zutreffend beschrieben sind.

## 3. Platzhalter im Text ausfüllen

Quelle der Texte: `docs/legal/datenschutz.md` und `docs/legal/impressum.md`. Nach jeder Änderung `node scripts/gen-legal.mjs` ausführen (siehe `docs/domains/privacy.md`).

- [ ] Verantwortlicher: `[Name, Anschrift, E-Mail]` (Datenschutz, Abschnitt 1 und 9)
- [ ] Name des Stalls: `[Name des Stalls]`
- [ ] `[Anbieter des Backup-Speichers, Standort EU]` (Abschnitte 5, 6)
- [ ] `[zuständige Aufsichtsbehörde, Anschrift, Website]` (Abschnitt 9)
- [ ] Hinweise `[Abschluss bestätigen]` und `[vor Veröffentlichung prüfen]` entfernen, sobald erledigt
- [ ] Impressum: Name, Anschrift, E-Mail, Telefon, Verbraucherstreitbeilegung
- [ ] Die Marke „Entwurf – vor Veröffentlichung rechtlich prüfen lassen“ aus beiden Texten entfernen, wenn die Prüfung abgeschlossen ist.
- [ ] Bei jeder inhaltlichen Änderung `TextVersion` in `backend/internal/privacy/consents.go` und die „Textversion“ in den Texten gemeinsam erhöhen (der Test `TestLegalTextsMatchTextVersion` schlägt sonst fehl).

## 4. Fragen für die rechtliche Prüfung

- [ ] Gilt die DSGVO hier, oder greift die Haushaltsausnahme (Art. 2 Abs. 2 lit. c)? Bei einer App für mehrere Einstaller, betrieben von einer Person, eher nicht.
- [ ] Impressumspflicht für ein privates, nicht geschäftsmäßiges Angebot.
- [ ] Rechtsgrundlagen je Datenkategorie (Einwilligung vs. Vertrag vs. berechtigtes Interesse), insbesondere Notfallkarte mit Telefonnummern für alle Mitglieder.
- [ ] Standortdaten (Geofence, GPS-Strecken): reicht die Einwilligung in der App, ist eine Datenschutz-Folgenabschätzung (Art. 35) nötig?
- [ ] Kartenkacheln von Apple, Google und OpenFreeMap (IP-Adresse und Kartenausschnitt des Betrachters): Die App holt vorher die Einwilligung „Karten anzeigen“ ein. Genügt das, und ist die Beschreibung in Abschnitt 3.4 vollständig?
- [ ] Minderjährige (Art. 8): Die App fragt nach dem Alter (Selbstauskunft „16 oder älter“, sonst Zustimmung eines Elternteils per E-Mail-Link, nur Hash gespeichert, Adresse des Elternteils als Nachweis). Genügt das als „angemessene Anstrengung“ (Art. 8 Abs. 2)? Gilt für den Nutzungsvertrag eines Minderjährigen zusätzlich §§ 107 ff. BGB?
- [ ] Reitbeteiligungen und Besitzer sind für alle Mitglieder sichtbar (Abschnitt 3.7): Vertrag oder berechtigtes Interesse?
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
- [ ] Datenschutz-Links in App Store Connect und Play Console eintragen (URL der Datenschutzerklärung, in der Web-App `https://stallfunk.de/legal/privacy`; die App zeigt den Text auch selbst an).
- [ ] App Store Connect „App-Datenschutz“ und Play Console „Data safety“ müssen zum Privacy Manifest in `mobile/app.json` (`ios.privacyManifests`: Name, E-Mail, Telefon, Nutzer-ID, genauer Standort, Fotos, Inhalte, Sensormerkmale; kein Tracking) und zum Datenschutztext passen. Hintergrundstandort in der Play Console begründen (Geofence, Streckenaufzeichnung).
- [ ] Export-Compliance: `ITSAppUsesNonExemptEncryption` ist `false` (nur Standard-TLS). Stimmt weiter, solange die App keine eigene Verschlüsselung mitbringt.
- [ ] Open-Source-Lizenzen: Die Hinweise erzeugt `mobile/scripts/gen-licenses.mjs` bei jedem `npm ci`; die Seite „Lizenzen“ in der App zeigt sie. Nichts mehr zu tun, außer ein Paket mit unzulässiger Lizenz bricht den Build ab.
- [ ] Anleitung für Eltern: Widerruf der Zustimmung läuft über die Adresse im Impressum (Konto wird dann gelöscht, `stallfunk-admin user delete`). Ablauf festlegen und Postfach im Blick behalten.
