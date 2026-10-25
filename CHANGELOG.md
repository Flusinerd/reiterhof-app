# Changelog

Alle nennenswerten Änderungen an Stallfunk. Versionen folgen [SemVer](https://semver.org/lang/de/),
die Einträge dem Format von [Keep a Changelog](https://keepachangelog.com/de/1.1.0/).

## [Unreleased]

### Neu

- **Übungsaufgaben**: Die App erstellt eine eigene Aufgabe für zu Hause (Klasse E, A, L oder L**, Viereck 20×40 oder
  20×60, kurz oder normal) mit Buchstaben, Weg und Zeit je Zeile. Bei Pferden mit Einheiten nur aus gefestigten und den
  nächsten Lektionen, auf Wunsch mit Schwerpunkt; „Neu würfeln“ liefert eine andere Aufgabe. Eine kurze Aufgabe hat
  höchstens 18, eine normale höchstens 26 Zeilen; Trab, Schritt mit freiem Schritt an der Diagonale und je ein
  Galoppblock auf jeder Hand folgen in fester Reihenfolge.
- **Küren**: Die Kür zeigt die Pflichtlektionen, die man aus dem eigenen Aufgabenheft einträgt (je Stall speicherbar),
  in der eingetragenen Zeit und prüft die Regeln des Kür-Merkblatts (Gruß, Schrittstrecke, Verstärkungen, Volten,
  Kurzkehrt, Lektionen höherer Klassen). Dazu Hinweise zur Choreografie und Musikteile je Gangart.
- **Bearbeiten**: Zeilen umsortieren oder entfernen; verletzt das eine Regel, nennt die App den Grund auf Deutsch.
- Die App liefert keine Aufgabentexte und keine Pflichtlektionen offizieller Prüfungen (Urheberrecht FNverlag).

### Behoben

- „Woche planen“ und das Speichern der Trainingsprofil-Seiten (Rhythmus, Aktivitäten, Reitbeteiligungen,
  Feste Tage, Turniere) schickten bei einem schnellen Doppeltipp zwei Anfragen; jetzt eine.
- Stall: Fiel „Wer ist da“ aus, stand dort dauerhaft „Wird geladen …“; jetzt eine Meldung, ein Tipp
  lädt neu.

## [0.3.0] – 2026-10-25

Die KI-Trainingsplanung ist neu aufgebaut: Sie folgt der Skala der Ausbildung, plant jede Einheit als
Lösen, Arbeiten und Entspannen und richtet sich nach Schwerpunkten, Ausstattung des Stalls und dem
Leistungsziel des Pferdes. Dazu sehen Bildschirme jetzt sofort, was man gerade bestätigt hat.

### Neu

- **Übungskatalog** mit 169 Übungen in sieben Bereichen (Dressur, Cavaletti, Springen, Bodenarbeit,
  Longe, Gelände, Western), jede mit Zweck, Phase, Dauer, Ausstattung und Quellen.
- **Schwerpunkte**: bis zu drei pro Pferd aus der Skala der Ausbildung und der Praxis, darunter
  „Turniervorbereitung“; Reitbeteiligungen setzen bis zu zwei eigene. Die Planung wechselt die
  Schwerpunkte ab und sagt, warum.
- **Stundenbild**: jede Einheit mit Lösen, Arbeiten und Entspannen samt Minuten und Übung;
  Aufbau- und Entlastungswochen, Wiederantrainieren nach Pausen, Grenzen für junge und alte Pferde.
- **Ausstattung des Stalls** je Ort, auch „geteilt“; pro Tag „Aufbau heute“. Ohne Stangen oder Aufbau
  plant die App eine gleichwertige Übung ohne und zeigt die Alternativen.
- **Leistungsziel**: Fortschritt zur nächsten Klasse mit den nächsten Lektionen und Vorschlag zum
  Höherstufen.
- **Übungsbibliothek** mit Filtern nach Schwerpunkt und Phase, Voraussetzungen, Alternativen und
  Quellen.

### Behoben

- „Erledigt“ bei Gesundheitsterminen verschob das Datum bei jedem weiteren Tipp; jetzt einmal pro Tag,
  mit Erfolgsanzeige und „Rückgängig“. Bei Medikamenten mit Enddatum heißt der Knopf „Gabe beenden“.
- Heute, Erinnerungen, Tagebuch und Pferdeliste zeigten nach Änderungen alte Daten; sie aktualisieren
  sich jetzt nach jeder Aktion, beim Zurückkehren und beim Öffnen der App.
- Knöpfe ließen sich während des Speicherns erneut antippen; Zeilen öffneten bei Doppeltipp doppelt.
- Ohne Netz behauptete Heute „alles erledigt“; jetzt mit Fehlerhinweis und „Erneut versuchen“.

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

[0.3.0]: https://github.com/flusinerd/reiterhof-app/releases/tag/v0.3.0
[0.2.0]: https://github.com/flusinerd/reiterhof-app/releases/tag/v0.2.0
[0.1.0]: https://github.com/flusinerd/reiterhof-app/commits/80a50a2
