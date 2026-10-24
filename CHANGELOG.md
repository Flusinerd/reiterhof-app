# Changelog

Alle nennenswerten Änderungen an Stallfunk. Versionen folgen [SemVer](https://semver.org/lang/de/),
die Einträge dem Format von [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).

## [0.2.0] – 2026-10-25

Die App ist neu gestaltet („Abendlicht“): Sie folgt der Hell-/Dunkel-Einstellung des Handys und ist
um den Tag der Besitzer:innen herum aufgebaut, nicht mehr um Fachbereiche.

### Neu

- **Heute**: Der Start zeigt die eine nächste Aufgabe mit ihrem Knopf und darunter die Zeitleiste für
  heute und morgen: Training, Turniere, die Decke heute Nacht, Termine und Medikamente der eigenen Pferde.
- **Neue Tab-Leiste**: Heute, das eigene Pferd (nach ihm benannt), Stall und Ich. Decken, Anfragen,
  Training und die Pferdeliste bleiben unter ihren bisherigen Adressen erreichbar.
- **Pferdeseite** mit optionalem Foto, „Heute für …“, Wochenleiste und Tagebuch aus Trainings und
  Beobachtungen; Gesundheit und Akte als Reiter.
- **Abendrunde**: Decken ein Pferd nach dem anderen abhaken, immer dunkel, mit Zusammenfassung;
  „Runde starten“ auf Heute, im Stall und bei den Decken.
- **„Wie war’s?“** nach dem Reiten: Dauer aus dem Tracking, Gefühl (Schwer / Gut / Sehr locker), Übung,
  Notiz für die Reitbeteiligung. Das Gefühl fließt in den KI-Wochenplan ein.
- **Stall**: Wer ist da, Decken heute Nacht als Tafel, Bitten um Hilfe als Nachrichten, Wetterwarnungen.
- **Wetterwarnungen** des DWD für den Standort des Stalls, mit Push bei Unwetter sowie bei Sturm,
  Frost, Glätte und Hitze.
- **Decke von Hand wählen**, abweichend von der Regel, für eine oder mehrere Nächte.
- **Gesundheitstermine**: Intervall in Tagen, Wochen, Monaten oder Jahren mit Vorschlägen je Art;
  Erinnerungstage und -uhrzeit pro Termin einstellbar.
- **Turniere** mit Startzeit und mehreren Prüfungen; am Turniertag plant die App keine Einheit.

### Behoben

- Foto-Upload: in der Web-App kam keine Datei an; iPhone-Fotos (HEIC) und Android-HEIF-Fotos wurden
  abgelehnt; hochgeladene Fotos standen seitlich.
- Schenkelweichen war in der Übungsbibliothek nicht über „Seitengänge“ zu finden und fehlte für
  Springpferde.
- Der Knopf „Eingedeckt“ war auf schmalen Handys abgeschnitten.
- Bilder mit Anmeldung (Decken, Beobachtungen, Dokumente) luden in der Web-App nicht.

### Geändert

- Alle übrigen Seiten (Anfragen, Anwesenheit, Erinnerungen, Training, Übungen, Gesundheit,
  Notfallkarte, Reha, Deckenplan, Einstellungen, Anmeldung) im neuen Stil: Zeilen mit Haarlinien statt
  Kartenstapeln, eine Hauptaktion pro Bildschirm, Fraunces-Kursiv für Titel.
- Die alten Chips „Frisch, Locker, Müde, Klemmig“ nach dem Training sind durch das Gefühl ersetzt;
  die Schnittstelle nimmt beides weiter an.

## [0.1.0]

Erste Version: Pferde, Decken mit Wetterregeln, Anfragen, Training mit Tracking und KI-Wochenplan,
Gesundheitstermine, Reha, Anwesenheit, Erinnerungen, Web-App.

[0.2.0]: https://github.com/flusinerd/reiterhof-app/releases/tag/v0.2.0
[0.1.0]: https://github.com/flusinerd/reiterhof-app/commits/80a50a2
