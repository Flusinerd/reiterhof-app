# Datenschutzerklärung

> **Entwurf – vor Veröffentlichung rechtlich prüfen lassen.** Diese Erklärung beschreibt, wie die App „Stallfunk“ heute technisch arbeitet. Die Platzhalter in eckigen Klammern müssen vor dem Einsatz ausgefüllt werden.

**Textversion:** 2026-09-30

## 1. Wer ist verantwortlich?

Verantwortlich für die Verarbeitung deiner Daten in dieser App ist:

[Name, Anschrift, E-Mail]

Die App ist ein privates Angebot für die Stallgasse [Name des Stalls]. Sie wird nicht gewerblich betrieben und enthält keine Werbung, kein Tracking zu Werbezwecken und keine Analyse-Dienste.

## 2. Das Wichtigste in Kürze

- Wir speichern nur, was die App für ihre Funktionen braucht.
- Dein Standort verlässt dein Handy nur, wenn du das Aufzeichnen einer Strecke beim Reiten selbst einschaltest. Die automatische Anmeldung im Stall (Geofence) läuft komplett auf dem Handy; der Server erfährt nur „angekommen“ und „gegangen“.
- Für Standort, Fotos, Benachrichtigungen und das Zeigen deiner Anwesenheit fragen wir dich vorher um Erlaubnis. Du kannst jede Erlaubnis jederzeit in der App zurücknehmen (Einstellungen, Datenschutz).
- Du kannst alle deine Daten in der App als Datei herunterladen und dein Konto selbst löschen.
- Die Daten liegen auf einem Server in Deutschland.

## 3. Welche Daten verarbeiten wir wozu?

### 3.1 Konto und Anmeldung

Daten: Name, E-Mail-Adresse, optional Telefonnummer, Avatar-Farbe, Zugehörigkeit zum Stall, ob du Admin bist, Zeitpunkt der Registrierung.

Zweck: Du brauchst ein Konto, damit die App weiß, wer du bist und was du sehen darfst.

Rechtsgrundlage: Durchführung des Nutzungsverhältnisses, Art. 6 Abs. 1 lit. b DSGVO.

Anmeldung per E-Mail-Link: Wir senden dir einen Link an deine Adresse. Der Link ist 15 Minuten gültig und funktioniert einmal. Gespeichert wird nur ein Hash des Links. Den Versand übernimmt der E-Mail-Dienst Resend (Resend, Inc., USA; Versand über die EU-Region). Resend verarbeitet dafür deine E-Mail-Adresse und den Inhalt der Mail.

Anmeldung mit Google oder Apple: Wenn du das wählst, sendet dein Handy ein Anmeldezeichen von Google bzw. Apple an unseren Server. Wir prüfen es und speichern die Kennung deines Kontos beim Anbieter („sub“), deine E-Mail-Adresse und ggf. deinen Namen. Wir erhalten kein Passwort. Für die Anmeldung gelten zusätzlich die Datenschutzhinweise von Google (Google Ireland Limited) bzw. Apple (Apple Distribution International Ltd.).

Sitzungen: Damit du angemeldet bleibst, speichern wir pro Gerät einen Sitzungs-Hash, die Gerätebezeichnung (User-Agent) und Zeitpunkte. Eine Sitzung endet 90 Tage nach der letzten Nutzung oder wenn du dich abmeldest.

### 3.2 Anwesenheit im Stall

Daten: Ankunft, Abgang und ob du dich von Hand oder automatisch (Geofence) angemeldet hast. Daraus berechnen wir „zuletzt gesehen“ und einen Hinweis, wann du meist kommst (Mittelwert der letzten 8 Wochen, nur bei Sichtbarkeit „alle“).

Wer sieht das? Das bestimmst du mit der Sichtbarkeit:

- **Alle:** Andere Mitglieder deines Stalls sehen, dass du da bist, seit wann und wann du zuletzt da warst.
- **Nur Tag:** Andere sehen, dass du heute da bist und an welchem Tag du zuletzt da warst, aber keine Uhrzeiten.
- **Niemand:** Andere sehen nichts. Das gilt auch für Admins.

Deine eigene Anwesenheit siehst du immer vollständig.

Zweck: Absprachen im Stall, zum Beispiel ob noch jemand da ist, bevor du gehst.

Rechtsgrundlage: Deine Einwilligung, Art. 6 Abs. 1 lit. a DSGVO. Die App fragt vor dem ersten Mal. Nimmst du sie zurück, wird deine Sichtbarkeit auf „Niemand“ gestellt.

Speicherdauer: Abgeschlossene Besuche werden nach 12 Monaten gelöscht.

### 3.3 Automatische Anmeldung im Stall (Geofence)

Wenn du „Automatisch erkennen“ einschaltest, überwacht dein Betriebssystem einen Bereich um den Stall (Standard 150 m). Betrittst oder verlässt du ihn, meldet die App dich an oder ab. Dazu braucht die App die Standortberechtigung „Immer“.

Dein Standort wird dabei nicht an den Server gesendet und nicht gespeichert. Der Server erfährt nur „angekommen“ bzw. „gegangen“ mit der Quelle „Geofence“. Die Einstellung gilt nur für dieses Gerät.

Rechtsgrundlage: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO. Ohne Einwilligung lehnt der Server automatische Anmeldungen ab.

### 3.4 Strecken aufzeichnen (GPS-Tracking beim Reiten)

Wenn du beim Reiten die Aufzeichnung einschaltest, wird deine Strecke (Positionspunkte mit Zeit, Strecke, Dauer und Gangarten) als Teil der Trainingseinheit auf dem Server gespeichert. Zusätzlich werden aus dem Bewegungssensor berechnete Merkmale je Zeitabschnitt (keine Positionsdaten) gespeichert, um die Gangarterkennung zu verbessern. Ohne Aufzeichnung wird kein Standort übertragen.

Zweck: Trainingsdokumentation für dich und für das Pferd (Belastung, Gangarten-Anteile).

Wer sieht das? Die Trainingseinheit sehen der Besitzer des Pferdes, die Reitbeteiligungen dieses Pferdes und Admins, soweit die Einheit nicht ausdrücklich verborgen ist.

Rechtsgrundlage: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO.

Speicherdauer: Die Positionspunkte und die Sensormerkmale werden nach 12 Monaten gelöscht. Die Einheit mit Dauer, Strecke und Gangarten-Anteilen bleibt als Trainingsverlauf des Pferdes erhalten.

### 3.5 Fotos und Dokumente

Daten: Fotos (Kamera oder Galerie) und Dokumente (Pferdepass, Impfpass, Versicherung als Foto oder PDF), die du hochlädst, sowie wer sie hochgeladen hat.

Speicherort: Dateien liegen auf dem Server, getrennt nach Stall. Fotos sind für Mitglieder deines Stalls abrufbar, Dokumente nur für den Besitzer, die Reitbeteiligungen des Pferdes und Admins.

Zweck: Pferdeakte, Auffälligkeiten dokumentieren, Decken erkennen.

Rechtsgrundlage: Einwilligung für den Zugriff auf Kamera und Galerie, Art. 6 Abs. 1 lit. a DSGVO; Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO, für die Speicherung der Dateien.

Hinweis: Fotos können Metadaten enthalten (zum Beispiel den Aufnahmeort). Achte darauf, keine Personen ohne deren Einverständnis zu fotografieren.

### 3.6 Notfallkarte und Telefonnummern

Zu jedem Pferd gibt es eine Notfallkarte mit Tierarzt, Versicherung, Medikamenten, Allergien und Notfallkontakten. Dazu zeigt die Karte die Telefonnummer des Besitzers, wenn dieser eine hinterlegt hat.

**Die Notfallkarte einschließlich der Telefonnummern ist für alle Mitglieder deines Stalls sichtbar.** Das ist Absicht: Im Notfall muss jeder anrufen können. Trage nur eine Nummer ein, wenn du damit einverstanden bist.

Trägst du als Besitzer weitere Kontaktpersonen (Name, Telefonnummer) ein, sorge bitte dafür, dass diese damit einverstanden sind.

Zweck: Schnelle Hilfe für das Tier im Notfall.

Rechtsgrundlage: Berechtigtes Interesse an schneller Hilfe im Notfall, Art. 6 Abs. 1 lit. f DSGVO, und deine Angabe der Nummer. Du kannst deine Nummer jederzeit im Profil löschen.

### 3.7 Pferdeakte und Gesundheitsdaten der Pferde

Angaben zu Pferden (Name, Box, Rasse, Gewicht, Termine wie Impfung und Hufschmied, Medikamente, Auffälligkeiten, Reha-Pläne, Trainingsprofil) sind zunächst Daten von Tieren. Sie werden aber personenbezogen, weil sie einem Besitzer zugeordnet sind und weil festgehalten wird, wer etwas gemeldet oder eingetragen hat (zum Beispiel „gemeldet von Anna“).

Zweck: Gemeinsame Versorgung der Pferde.

Rechtsgrundlage: Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO, und berechtigtes Interesse an guter Versorgung der Tiere, Art. 6 Abs. 1 lit. f DSGVO.

### 3.8 Anfragen, Decken- und Trainingsplan

Wenn du eine Anfrage stellst (zum Beispiel „Hilfe beim Decken“) oder hilfst, sehen die Mitglieder des Stalls deinen Namen dazu. Das Gleiche gilt für eingetragene Wochenplan-Plätze, Deckenwechsel und Trainingseinheiten, die du einträgst.

Rechtsgrundlage: Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO.

### 3.9 Benachrichtigungen (Push)

Damit wir dich erinnern können (Termine, Anfragen, Decken), sendet dein Handy einen Push-Token an unseren Server. Wir speichern ihn mit der Plattform (iOS oder Android). Nachrichten werden über den Push-Dienst von Expo (Expo, 650 Industries, Inc., USA) und die Dienste von Apple (APNs) bzw. Google (Firebase Cloud Messaging) zugestellt. Der Nachrichtentext läuft dabei durch diese Dienste.

Rechtsgrundlage: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO. Widerrufst du sie, löschen wir deine Push-Tokens. Einzelne Erinnerungsarten kannst du zusätzlich abschalten.

Drittlandübermittlung: Expo, Resend und ggf. Google und Apple verarbeiten Daten auch in den USA. Grundlage sind Angemessenheitsbeschluss (EU-US Data Privacy Framework) bzw. Standardvertragsklauseln des jeweiligen Anbieters [vor Veröffentlichung prüfen].

### 3.10 Wetter

Die Wetterdaten stammen vom Deutschen Wetterdienst (Open Data). Der Abruf erfolgt durch unseren Server für den Standort des Stalls. Es werden keine personenbezogenen Daten von dir an den Deutschen Wetterdienst gesendet; dort ist nur die IP-Adresse unseres Servers sichtbar.

## 4. Wo laufen die Daten? Hosting und Backups

- **Hosting:** Server der Netcup GmbH, Karlsruhe, Rechenzentrum in Deutschland. Mit Netcup besteht ein Vertrag zur Auftragsverarbeitung nach Art. 28 DSGVO [Abschluss bestätigen].
- **Server-Protokolle:** Der Webserver schreibt kein Zugriffsprotokoll mit IP-Adressen. Die API protokolliert Fehler ohne Inhalte deiner Einträge. Für den Schutz vor zu vielen Anmeldeversuchen wird deine IP-Adresse kurz im Arbeitsspeicher gehalten und nicht gespeichert.
- **Backups:** Täglich wird die Datenbank gesichert und zusammen mit den hochgeladenen Dateien verschlüsselt an einen externen Speicher [Anbieter des Backup-Speichers, Standort EU] kopiert (Auftragsverarbeitung nach Art. 28 DSGVO [Abschluss bestätigen]). Backups werden auf dem Server bis zu 14 Tage (tägliche) bzw. 8 Wochen (wöchentliche) und extern bis zu 90 Tage aufbewahrt. Gelöschte Daten verschwinden aus Backups daher spätestens nach 90 Tagen. Wird ein Backup zurückgespielt, werden Löschungen erneut ausgeführt.

## 5. Wer bekommt deine Daten?

- Mitglieder deines Stalls, soweit oben beschrieben und nach deinen Sichtbarkeits-Einstellungen.
- Netcup GmbH als Hosting-Anbieter (Auftragsverarbeiter).
- Anbieter des externen Backup-Speichers (Auftragsverarbeiter).
- Resend, Inc. (USA) als E-Mail-Dienst für den Versand der Anmelde-Links (Auftragsverarbeiter).
- Expo, Apple und Google für Push-Benachrichtigungen, und Google bzw. Apple, wenn du dich damit anmeldest.
- Deutscher Wetterdienst (nur Abruf öffentlicher Daten, keine Übermittlung von Personendaten).

Wir verkaufen keine Daten und geben sie nicht zu Werbezwecken weiter.

## 6. Wie lange speichern wir?

- **Konto:** bis du es löschst.
- **Sitzungen:** 90 Tage nach der letzten Nutzung; abgelaufene Sitzungen und Anmelde-Links werden täglich gelöscht.
- **Anwesenheits-Besuche:** 12 Monate.
- **Positionspunkte aufgezeichneter Strecken:** 12 Monate.
- **Versendete Erinnerungen:** 12 Monate.
- **Einwilligungen:** solange das Konto besteht (als Nachweis).
- **Pferdeakte, Dokumente, Anfragen, Trainingseinheiten:** solange sie gebraucht werden; sie können vom Besitzer oder Admin gelöscht werden. Beim Löschen deines Kontos bleiben Einträge, die andere betreffen, ohne deinen Namen erhalten (siehe 7).
- **Backups:** bis zu 90 Tage (siehe 4).

## 7. Was passiert, wenn ich mein Konto lösche?

In der App unter Einstellungen, Datenschutz, „Konto löschen“. Dann gilt:

- Gelöscht werden: Anmeldungen und Sitzungen, Push-Tokens, Erinnerungs-Einstellungen, Anwesenheits-Besuche, Reitbeteiligungen, Hilfe-Zusagen, Einwilligungen, aufgezeichnete Strecken und Fotos deiner Meldungen.
- Dein Name, deine E-Mail-Adresse, Telefonnummer und Farbe werden entfernt. Einträge, die für die Pferde wichtig bleiben (zum Beispiel gemeldete Auffälligkeiten und Trainingseinheiten), bleiben ohne Bezug zu dir bestehen und werden als „Gelöschtes Mitglied“ angezeigt. Deine offenen Anfragen werden abgesagt.
- Besitzt du Pferde, musst du sie vorher einem anderen Mitglied übergeben oder löschen lassen. Der letzte Admin eines Stalls muss vorher einen Nachfolger bestimmen.

## 8. Deine Rechte

Du hast das Recht auf

- Auskunft (Art. 15 DSGVO) und Datenübertragbarkeit (Art. 20): In der App unter Einstellungen, Datenschutz, „Meine Daten herunterladen“ erhältst du alle deine Daten als JSON-Datei.
- Berichtigung (Art. 16): Profildaten kannst du selbst ändern.
- Löschung (Art. 17): siehe Punkt 7.
- Einschränkung der Verarbeitung (Art. 18) und Widerspruch (Art. 21) gegen Verarbeitungen, die auf berechtigtem Interesse beruhen.
- Widerruf erteilter Einwilligungen mit Wirkung für die Zukunft (Art. 7 Abs. 3): Einstellungen, Datenschutz. Die Rechtmäßigkeit der bisherigen Verarbeitung bleibt davon unberührt.

Schreibe uns für alles andere an [Name, Anschrift, E-Mail].

**Beschwerderecht:** Du kannst dich bei einer Datenschutz-Aufsichtsbehörde beschweren, zum Beispiel bei [zuständige Aufsichtsbehörde, Anschrift, Website].

## 9. Automatisierte Entscheidungen

Es gibt keine automatisierten Entscheidungen mit rechtlicher Wirkung und kein Profiling. Empfehlungen wie „Was heute?“ und die Deckenempfehlung folgen festen Regeln, sind Vorschläge und betreffen Pferde, nicht dich.

## 10. Sicherheit

Die Verbindung zur App ist per TLS verschlüsselt. Anmelde-Links und Sitzungs-Schlüssel werden nur als Hash gespeichert. Auf dem Handy liegt der Sitzungs-Schlüssel im geschützten Speicher (Keychain bzw. Keystore). Der Server ist gehärtet und wird automatisch mit Sicherheitsupdates versorgt.

## 11. Änderungen

Ändert sich diese Erklärung, erhöhen wir die Textversion und fragen erteilte Einwilligungen bei Bedarf erneut ab. Die aktuelle Fassung findest du in der App unter Einstellungen, Datenschutz.
