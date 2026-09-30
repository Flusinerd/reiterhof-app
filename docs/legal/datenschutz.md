# Datenschutzerklärung

**Textversion:** 2026-09-30.2

## 1. Wer ist verantwortlich?

Verantwortlich für die Verarbeitung deiner Daten in dieser App ist:

Jan Krüger, E-Mail: jankrueger1999@gmail.com

Die App ist ein privates Angebot einer Privatperson, die selbst Einstaller ist, für die Einstaller ihres Stalls. Sie ist nur auf Einladung nutzbar, wird nicht gewerblich betrieben und enthält keine Werbung, kein Tracking zu Werbezwecken und keine Analyse-Dienste. Der Stallbetrieb betreibt die App nicht und entscheidet nicht darüber, welche Daten sie verarbeitet.

## 2. Das Wichtigste in Kürze

- Wir speichern nur, was die App für ihre Funktionen braucht.
- Dein Standort verlässt dein Handy nur, wenn du das Aufzeichnen einer Strecke beim Reiten selbst einschaltest. Die automatische Anmeldung im Stall (Geofence) läuft komplett auf dem Handy; der Server erfährt nur „angekommen“ und „gegangen“.
- Für Standort, Karten, Fotos, Benachrichtigungen, das Zeigen deiner Anwesenheit und KI-Vorschläge im Wochenplan fragen wir dich vorher um Erlaubnis. Du kannst jede Erlaubnis jederzeit in der App zurücknehmen (Einstellungen, Datenschutz).
- Fotos werden beim Hochladen von Metadaten wie dem Aufnahmeort befreit.
- Wer jünger als 16 ist, braucht die Zustimmung eines Elternteils (Punkt 4).
- Du kannst alle deine Daten in der App als Datei herunterladen und dein Konto selbst löschen.
- Die Daten liegen auf einem Server in Deutschland.
- Den Wochenplan eines Pferdes kann auf Wunsch des Besitzers eine KI von Mistral AI (Frankreich) vorschlagen. Sie bekommt nur Trainingsdaten des Pferdes ohne Namen und nutzt sie nicht zum Training ihrer Modelle (Punkt 3.12).

## 3. Welche Daten verarbeiten wir wozu?

### 3.1 Konto und Anmeldung

Daten: Name, E-Mail-Adresse, optional Telefonnummer, Avatar-Farbe, Zugehörigkeit zum Stall, ob du Admin bist, Zeitpunkt der Registrierung, deine Altersangabe (Punkt 4).

Zweck: Du brauchst ein Konto, damit die App weiß, wer du bist und was du sehen darfst.

Rechtsgrundlage: Durchführung des Nutzungsverhältnisses, Art. 6 Abs. 1 lit. b DSGVO.

Anmeldung per E-Mail: Wir senden dir eine Mail mit einem Link und einem 6-stelligen Code (der Code steht auch in der Betreffzeile). Beide gelten 15 Minuten und funktionieren einmal. Gespeichert werden nur ein Hash des Links und ein mit einem Server-Schlüssel gesicherter Hash des Codes; abgelaufene Anmeldeversuche werden nach spätestens einem Tag gelöscht. Den Versand übernimmt der E-Mail-Dienst Resend (Plus Five Five, Inc., San Francisco, USA; die Mails werden über die EU-Region versendet, Konto- und Nutzungsdaten des Dienstes liegen in den USA). Resend verarbeitet dafür deine E-Mail-Adresse und den Inhalt der Mail; mit Resend besteht ein Vertrag zur Auftragsverarbeitung mit EU-Standardvertragsklauseln, außerdem ist Resend nach dem EU-US Data Privacy Framework zertifiziert.

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

### 3.4 Strecken aufzeichnen (GPS-Tracking beim Reiten) und Karten

Wenn du beim Reiten die Aufzeichnung einschaltest, wird deine Strecke (Positionspunkte mit Zeit, Strecke, Dauer und Gangarten) als Teil der Trainingseinheit auf dem Server gespeichert. Zusätzlich werden aus dem Bewegungssensor berechnete Merkmale je Zeitabschnitt (keine Positionsdaten) gespeichert, um die Gangarterkennung zu verbessern. Ohne Aufzeichnung wird kein Standort übertragen.

Zweck: Trainingsdokumentation für dich und für das Pferd (Belastung, Gangarten-Anteile).

Wer sieht das? Die Trainingseinheit sehen der Besitzer des Pferdes, die Reitbeteiligungen dieses Pferdes und Admins, soweit die Einheit nicht ausdrücklich verborgen ist.

Rechtsgrundlage: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO.

Speicherdauer: Die Positionspunkte und die Sensormerkmale werden nach 12 Monaten gelöscht. Die Einheit mit Dauer, Strecke und Gangarten-Anteilen bleibt als Trainingsverlauf des Pferdes erhalten.

Kartenanzeige: Die Karte zur Strecke lädt Kartenkacheln von einem Kartendienst: auf dem iPhone von Apple (Apple Maps), auf Android von Google (Google Maps) und in der Web-App von OpenFreeMap (Kartendaten der OpenStreetMap-Gemeinschaft). Der jeweilige Dienst sieht dabei deine IP-Adresse, Angaben deines Geräts und den Kartenausschnitt, also die Gegend deiner Strecke. Unser Server ist daran nicht beteiligt. Die Karte wird erst geladen, wenn du das in der App erlaubst („Karten anzeigen“); ohne Karte werden Strecke, Tempo und Gangarten trotzdem aufgezeichnet.

Rechtsgrundlage für die Kartenanzeige: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO. Google und Apple verarbeiten Daten auch in den USA (Punkt 3.9, Drittlandübermittlung).

### 3.5 Fotos und Dokumente

Daten: Fotos (Kamera oder Galerie) und Dokumente (Pferdepass, Impfpass, Versicherung als Foto oder PDF), die du hochlädst, sowie wer sie hochgeladen hat.

Metadaten: Beim Hochladen entfernt der Server aus JPEG-, PNG- und WebP-Bildern die eingebetteten Metadaten (zum Beispiel Aufnahmeort, Kamera, Zeitpunkt, Kommentare, eingebettete Vorschaubilder und Anhänge). Bilder in anderen Formaten nimmt die App nicht an. PDF-Dokumente werden unverändert gespeichert; sie können Angaben wie den Autor enthalten.

Speicherort: Dateien liegen auf dem Server, getrennt nach Stall. Fotos zu Auffälligkeiten und Decken sind für Mitglieder deines Stalls abrufbar, Dokumente nur für den Besitzer, die Reitbeteiligungen des Pferdes und Admins. Der Server gibt eine Datei nur heraus, wenn der Eintrag, zu dem sie gehört, für dich sichtbar ist.

Zweck: Pferdeakte, Auffälligkeiten dokumentieren, Decken erkennen.

Rechtsgrundlage: Einwilligung für den Zugriff auf Kamera und Galerie, Art. 6 Abs. 1 lit. a DSGVO; Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO, für die Speicherung der Dateien.

Hinweis: Achte darauf, keine Personen ohne deren Einverständnis zu fotografieren.

### 3.6 Notfallkarte und Telefonnummern

Zu jedem Pferd gibt es eine Notfallkarte mit Tierarzt, Versicherung, Medikamenten, Allergien und Notfallkontakten. Dazu zeigt die Karte die Telefonnummer des Besitzers, wenn dieser eine hinterlegt hat.

**Die Notfallkarte einschließlich der Telefonnummern ist für alle Mitglieder deines Stalls sichtbar.** Das ist Absicht: Im Notfall muss jeder anrufen können. Trage nur eine Nummer ein, wenn du damit einverstanden bist.

Trägst du als Besitzer weitere Kontaktpersonen (Name, Telefonnummer) ein, sorge bitte dafür, dass diese damit einverstanden sind.

Zweck: Schnelle Hilfe für das Tier im Notfall.

Rechtsgrundlage: Berechtigtes Interesse an schneller Hilfe im Notfall, Art. 6 Abs. 1 lit. f DSGVO, und deine Angabe der Nummer. Du kannst deine Nummer jederzeit im Profil löschen.

### 3.7 Pferdeakte und Gesundheitsdaten der Pferde

Angaben zu Pferden (Name, Box, Rasse, Gewicht, Termine wie Impfung und Hufschmied, Medikamente, Auffälligkeiten, Reha-Pläne, Trainingsprofil) sind zunächst Daten von Tieren. Sie werden aber personenbezogen, weil sie einem Besitzer zugeordnet sind und weil festgehalten wird, wer etwas gemeldet oder eingetragen hat (zum Beispiel „gemeldet von Anna“).

Alle Mitglieder deines Stalls sehen, wer Besitzer eines Pferdes ist und wer als Reitbeteiligung welche Rechte an einem Pferd hat.

Zweck: Gemeinsame Versorgung der Pferde.

Rechtsgrundlage: Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO, und berechtigtes Interesse an guter Versorgung der Tiere, Art. 6 Abs. 1 lit. f DSGVO.

### 3.8 Anfragen, Decken- und Trainingsplan

Wenn du eine Anfrage stellst (zum Beispiel „Hilfe beim Decken“) oder hilfst, sehen die Mitglieder des Stalls deinen Namen dazu. Das Gleiche gilt für eingetragene Wochenplan-Plätze, Deckenwechsel und Trainingseinheiten, die du einträgst. Erzeugst du als Admin Einladungscodes, wird festgehalten, dass du sie erzeugt hast.

Rechtsgrundlage: Nutzungsverhältnis, Art. 6 Abs. 1 lit. b DSGVO.

### 3.9 Benachrichtigungen (Push)

Damit wir dich erinnern können (Termine, Anfragen, Decken), sendet dein Handy einen Push-Token an unseren Server. Wir speichern ihn mit der Plattform (iOS oder Android). Nachrichten schickt unser Server direkt an den Push-Dienst deines Geräteherstellers: auf dem iPhone an Apple (Apple Push Notification service), auf Android an Google (Firebase Cloud Messaging). Ein weiterer Dienst ist nicht beteiligt. Der Nachrichtentext (zum Beispiel „Balu braucht heute eine Decke“) läuft dabei durch den Dienst von Apple bzw. Google.

Nutzt du Stallfunk als Web-App (zum Home-Bildschirm hinzugefügt), speichern wir stattdessen die Push-Adresse deines Browsers mit zwei Schlüsseln und die Browserkennung (User-Agent). Nachrichten werden verschlüsselt an den Push-Dienst des Browser-Herstellers geschickt (bei Safari auf dem iPhone Apple, bei Chrome Google, bei Firefox Mozilla) und erst auf deinem Gerät entschlüsselt.

Rechtsgrundlage: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO. Widerrufst du sie, löschen wir deine Push-Tokens und Browser-Push-Adressen. Einzelne Erinnerungsarten kannst du zusätzlich abschalten.

Drittlandübermittlung: Resend und ggf. Google und Apple (Anmeldung, Push, Karten) verarbeiten Daten auch in den USA. Grundlage sind der Angemessenheitsbeschluss zum EU-US Data Privacy Framework bzw. Standardvertragsklauseln des jeweiligen Anbieters; für Resend ist beides vereinbart. Google (Google LLC) ist nach dem EU-US Data Privacy Framework zertifiziert; Apple stützt die Übermittlung in die USA auf die Standardvertragsklauseln der EU-Kommission.

### 3.10 Wetter

Die Wetterdaten stammen vom Deutschen Wetterdienst (Open Data). Der Abruf erfolgt durch unseren Server für den Standort des Stalls. Es werden keine personenbezogenen Daten von dir an den Deutschen Wetterdienst gesendet; dort ist nur die IP-Adresse unseres Servers sichtbar.

### 3.11 Berechtigungen und Speicher auf deinem Gerät

Die App fragt das Betriebssystem erst um eine Berechtigung, wenn du die Funktion nutzt, und nur nach deiner Erlaubnis in der App: Standort (Geofence, Streckenaufzeichnung), Bewegungssensor (Gangarterkennung, die Daten bleiben bis auf die Merkmale aus Punkt 3.4 auf dem Gerät), Kamera und Fotos, Mitteilungen. Für Kalendereinträge nutzt die App den Eintragsdialog deines Geräts; sie liest deinen Kalender nicht. Mikrofon, Kontakte oder Werbe-Kennungen nutzt die App nicht.

Auf deinem Gerät speichert die App nur, was sie zum Funktionieren braucht: den Sitzungs-Schlüssel (in der App im geschützten Speicher des Betriebssystems, Keychain bzw. Keystore; in der Web-App im Browserspeicher „localStorage“), die Einstellung der automatischen Anmeldung und ob dich die App schon nach einer Erlaubnis gefragt hat. Die Web-App richtet außerdem einen Service Worker für Mitteilungen ein. Es gibt keine Cookies zu Werbe- oder Analysezwecken und keine Zählpixel. Diese Speicherung ist für den Dienst unbedingt erforderlich (§ 25 Abs. 2 TDDDG) und wird beim Abmelden gelöscht.

### 3.12 Wochenplan mit KI-Vorschlägen (Mistral AI)

Besitzer und Admins können sich für ein Pferd einen Vorschlag für die offenen Tage der Woche machen lassen („Woche planen“). Hat der Besitzer des Pferdes „KI-Vorschläge für den Wochenplan“ erlaubt, fragt unser Server dafür ein Sprachmodell der Mistral AI SAS (Paris, Frankreich). Ohne diese Erlaubnis kommt der Vorschlag nur aus den festen Regeln der App, und es wird nichts an Mistral gesendet.

Gesendet werden nur Trainingsdaten des Pferdes: Disziplin, Trainingsstatus (fit, Reha, Pause), der gewünschte Rhythmus, die freigegebenen Aktivitäten, die Trainingseinheiten der letzten 14 Tage (vor wie vielen Tagen, Aktivität, Minuten, Belastung, Galopp-Anteil, wie sich das Pferd angefühlt hat), in wie vielen Tagen Turniere sind, die Vorgaben eines Reha-Plans je Tag (Aktivität und Minuten), Wetter und Boden von heute und welche Tage der Woche schon vergeben sind.

Nicht gesendet werden Namen von Personen oder Pferden, Kennungen, Notizen und andere Freitexte (auch keine Turniernamen, Diagnosen oder Reha-Bedingungen), Kalenderdaten und Standortdaten. Wer eine Einheit geritten hat, erfährt Mistral nicht.

Die festen Regeln der App prüfen jeden Vorschlag (Profil, Reha-Plan, Ruhetag nach dem Turnier, Rhythmus) und ersetzen, was nicht passt. Gespeichert wird der Vorschlag erst, wenn du ihn übernimmst; im Wochenplan steht dann „KI-Vorschlag“ dabei.

Mistral verarbeitet die Anfragen als Auftragsverarbeiter in unserem Auftrag. In unserem Konto bei Mistral ist ausgeschaltet, dass Anfragen zum Training der Modelle genutzt werden, und wir verwenden keine Modelle, bei denen Mistral trotzdem trainiert („Labs“). Mistral speichert Anfragen laut eigenen Angaben bis zu 30 Tage, um Missbrauch zu erkennen, und löscht sie danach. Unser Server speichert weder die Anfrage noch die Antwort.

Rechtsgrundlage: Einwilligung des Besitzers, Art. 6 Abs. 1 lit. a DSGVO. Einheiten, die Reitbeteiligungen eingetragen haben, gehen ohne Namen und ohne Datum mit in die Anfrage ein; Rechtsgrundlage dafür ist das berechtigte Interesse an einer guten Trainingsplanung für das Pferd, Art. 6 Abs. 1 lit. f DSGVO. Die Einwilligung kannst du jederzeit in den Einstellungen unter Datenschutz zurücknehmen; danach wird nichts mehr an Mistral gesendet.

## 4. Mindestalter und Zustimmung der Eltern

Standort, Karten, Fotos, Mitteilungen, das Zeigen deiner Anwesenheit und KI-Vorschläge im Wochenplan beruhen auf deiner Einwilligung. Nach Art. 8 DSGVO kann in Deutschland nur einwilligen, wer mindestens 16 Jahre alt ist; jüngere Personen brauchen die Zustimmung eines Elternteils oder einer erziehungsberechtigten Person.

Deshalb fragt die App dich beim ersten Start nach deinem Alter. Wir speichern kein Geburtsdatum, nur deine Angabe „16 oder älter“ mit Zeitpunkt. Bist du jünger, nennst du die E-Mail-Adresse eines Elternteils. Diese Person erhält eine Mail (Versand über Resend, Punkt 3.1) mit deinem Namen, deiner E-Mail-Adresse, einer Beschreibung dessen, was die App speichert, und einem Link. Über den Link stimmt sie zu; der Link gilt 7 Tage und funktioniert einmal, gespeichert wird nur ein Hash. Bis zur Zustimmung kannst du keinem Stall beitreten und keine Erlaubnis erteilen; dein Konto kannst du in dieser Zeit ansehen und löschen.

Wir speichern die E-Mail-Adresse des Elternteils und den Zeitpunkt der Zustimmung als Nachweis, solange dein Konto besteht. Ein Elternteil kann die Zustimmung jederzeit widerrufen, indem es sich an die in Punkt 1 genannte Adresse wendet; dann wird das Konto gelöscht.

Rechtsgrundlage: Art. 8 DSGVO in Verbindung mit Art. 6 Abs. 1 lit. c DSGVO (Nachweis der Zustimmung).

## 5. Wo laufen die Daten? Hosting und Backups

- **Hosting:** Server der Netcup GmbH, Karlsruhe, Rechenzentrum in Deutschland. Mit Netcup besteht ein Vertrag zur Auftragsverarbeitung nach Art. 28 DSGVO; Netcup setzt dafür Gesellschaften der Anexia-Gruppe in Deutschland und Österreich ein.
- **Server-Protokolle:** Der Webserver schreibt kein Zugriffsprotokoll mit IP-Adressen. Die API protokolliert Fehler ohne Inhalte deiner Einträge; die Datenbank protokolliert langsame Abfragen ohne die abgefragten Werte. Fehlerprotokolle werden nach vier Wochen gelöscht. Für den Schutz vor zu vielen Anmeldeversuchen wird deine IP-Adresse bis zu 24 Stunden im Arbeitsspeicher gehalten und nicht dauerhaft gespeichert.
- **Backups:** Täglich werden die Datenbank und die hochgeladenen Dateien auf demselben Server gesichert; die Sicherungen verlassen den Server nicht. Aufbewahrt werden 14 tägliche und 8 wöchentliche Stände. Gelöschte Daten verschwinden aus den Backups daher spätestens nach 8 Wochen. Wird ein Backup zurückgespielt, werden Löschungen erneut ausgeführt.

## 6. Wer bekommt deine Daten?

- Mitglieder deines Stalls, soweit oben beschrieben und nach deinen Sichtbarkeits-Einstellungen.
- Netcup GmbH als Hosting-Anbieter (Auftragsverarbeiter).
- Resend (Plus Five Five, Inc., USA) als E-Mail-Dienst für Anmelde-Mails und die Mail an Eltern (Auftragsverarbeiter).
- Apple und Google für Push-Benachrichtigungen (in der Web-App die Push-Dienste von Apple, Google oder Mozilla, je nach Browser), und Google bzw. Apple, wenn du dich damit anmeldest.
- Apple, Google oder OpenFreeMap als Kartendienst, wenn du Karten erlaubst (Punkt 3.4).
- Mistral AI SAS (Paris, Frankreich) für KI-Vorschläge im Wochenplan, wenn der Besitzer des Pferdes sie erlaubt (Auftragsverarbeiter, Punkt 3.12).
- Deutscher Wetterdienst (nur Abruf öffentlicher Daten, keine Übermittlung von Personendaten).
- Das Elternteil, das du für die Zustimmung nennst, erfährt deinen Namen und deine E-Mail-Adresse (Punkt 4).

Wir verkaufen keine Daten und geben sie nicht zu Werbezwecken weiter.

## 7. Wie lange speichern wir?

- **Konto:** bis du es löschst.
- **Sitzungen:** 90 Tage nach der letzten Nutzung; abgelaufene Sitzungen und Anmeldeversuche werden täglich gelöscht.
- **Anwesenheits-Besuche:** 12 Monate.
- **Positionspunkte aufgezeichneter Strecken:** 12 Monate.
- **Versendete Erinnerungen:** 12 Monate.
- **Einladungscodes:** 30 Tage nach Ablauf.
- **Links an Eltern:** 7 Tage; die Adresse des Elternteils und die Zustimmung, solange das Konto besteht.
- **Einwilligungen und Altersangabe:** solange das Konto besteht (als Nachweis).
- **Pferdeakte, Dokumente, Anfragen, Trainingseinheiten:** solange sie gebraucht werden; sie können vom Besitzer oder Admin gelöscht werden. Beim Löschen deines Kontos bleiben Einträge, die andere betreffen, ohne deinen Namen erhalten (siehe 8).
- **Anfragen für KI-Vorschläge:** bei uns gar nicht; bei Mistral bis zu 30 Tage (Punkt 3.12). Übernommene Vorschläge bleiben im Wochenplan wie andere Einträge.
- **Fehlerprotokolle des Servers:** 4 Wochen.
- **Backups:** bis zu 8 Wochen (siehe 5).

## 8. Was passiert, wenn ich mein Konto lösche?

In der App unter Einstellungen, Datenschutz, „Konto löschen“. Dann gilt:

- Gelöscht werden: Anmeldungen und Sitzungen, Push-Tokens und Browser-Push-Adressen, Erinnerungs-Einstellungen, Anwesenheits-Besuche, Reitbeteiligungen, Hilfe-Zusagen, Einwilligungen, Altersangabe und Adresse des Elternteils, aufgezeichnete Strecken und Sensormerkmale sowie Fotos deiner Meldungen.
- Dein Name, deine E-Mail-Adresse, Telefonnummer und Farbe werden entfernt. Einträge, die für die Pferde wichtig bleiben (zum Beispiel gemeldete Auffälligkeiten und Trainingseinheiten), bleiben ohne Bezug zu dir bestehen und werden als „Gelöschtes Mitglied“ angezeigt. Deine offenen Anfragen werden abgesagt.
- Besitzt du Pferde, musst du sie vorher einem anderen Mitglied übergeben oder löschen lassen. Der letzte Admin eines Stalls muss vorher einen Nachfolger bestimmen.

## 9. Deine Rechte

Du hast das Recht auf

- Auskunft (Art. 15 DSGVO) und Datenübertragbarkeit (Art. 20): In der App unter Einstellungen, Datenschutz, „Daten exportieren“ erhältst du alle deine Daten als JSON-Datei.
- Berichtigung (Art. 16): Profildaten kannst du selbst ändern.
- Löschung (Art. 17): siehe Punkt 8.
- Einschränkung der Verarbeitung (Art. 18) und Widerspruch (Art. 21) gegen Verarbeitungen, die auf berechtigtem Interesse beruhen.
- Widerruf erteilter Einwilligungen mit Wirkung für die Zukunft (Art. 7 Abs. 3): Einstellungen, Datenschutz. Die Rechtmäßigkeit der bisherigen Verarbeitung bleibt davon unberührt.

Schreibe für alles andere an die E-Mail-Adresse aus Punkt 1.

**Beschwerderecht:** Du kannst dich bei einer Datenschutz-Aufsichtsbehörde beschweren (Art. 77 DSGVO), zum Beispiel bei der Behörde des Bundeslandes, in dem du wohnst. Die Anschriften aller Aufsichtsbehörden stehen auf der Website der Bundesbeauftragten für den Datenschutz und die Informationsfreiheit, www.bfdi.bund.de.

## 10. Automatisierte Entscheidungen

Es gibt keine automatisierten Entscheidungen mit rechtlicher Wirkung und kein Profiling. Empfehlungen wie „Was heute?“ und die Deckenempfehlung folgen festen Regeln. Den Wochenplan kann zusätzlich ein Sprachmodell vorschlagen (Punkt 3.12); die festen Regeln prüfen jeden dieser Vorschläge. Alles sind Vorschläge, die Pferde betreffen, nicht dich; eingetragen wird nur, was jemand übernimmt.

## 11. Sicherheit

Die Verbindung zur App ist per TLS verschlüsselt. Anmelde-Links, Codes und Sitzungs-Schlüssel werden nur als Hash gespeichert. In der App liegt der Sitzungs-Schlüssel im geschützten Speicher des Betriebssystems (Keychain bzw. Keystore), in der Web-App im Browserspeicher; der Schlüssel wird nie in eine Web-Adresse geschrieben, zum Öffnen einer Datei erzeugt der Server stattdessen einen Link, der fünf Minuten gilt und nur diese Datei öffnet. Der Server ist gehärtet und wird automatisch mit Sicherheitsupdates versorgt.

## 12. Änderungen

Ändert sich diese Erklärung, erhöhen wir die Textversion und fragen erteilte Einwilligungen bei Bedarf erneut ab. Die aktuelle Fassung findest du in der App unter Einstellungen, Datenschutz.
