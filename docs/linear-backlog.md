# Linear-Backlog: noch anzulegende Tickets

Projekt: **Reiterhof App** (`P-JAN-1`), Team „Jan K“. Quelle: „Stallgasse – App Specification“.

Bereits angelegt: JAN-5 bis JAN-24 (Grundlagen, Anwesenheit manuell), dazu die Meilensteine M1 bis M8.
Diese Datei listet alle Tickets, die noch fehlen. Priorität: 1 dringend, 2 hoch, 3 mittel, 4 niedrig.

Zusätzlich: In JAN-22 ist „CLAUDE.md/README-Konvention“ von Linear fälschlich in einen Link umgewandelt worden. Den Text auf „CLAUDE.md- und README-Konvention“ (ohne Schrägstrich) korrigieren.

## M2 Anwesenheit

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 2 | Sichtbarkeit der Anwesenheit (alle / nur Tag / versteckt) | Pro Person `presence_visibility`: `all`, `only_day` (nur Datum, keine Uhrzeit), `hidden`. Standard `all`. Durchsetzung serverseitig (RLS/Views), nicht nur in der App. |
| 3 | Screen Anwesenheit (`/presence`) | Hero: mein Status (Ich gehe), Geofence-Schalter, Sichtbarkeit. „Jetzt da“ als Avatar-Kacheln mit „seit“, „Zuletzt gesehen“ je Person. Optional Hinweis „kommt meist gegen 19 Uhr“, aus der Historie berechnet. |
| 3 | Automatisches Ein-/Auschecken per Geofence (Opt-in pro Gerät) | Geofence um `stables.lat/lng` mit `expo-location`, `source = geofence`. Radius und Akkustrategie (Significant-Location vs. Region-Monitoring) sind offene Entscheidungen. Braucht die Datenschutz-Einwilligung (M8). |

## M3 Decken und Wetter

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 2 | Wetterabruf und Snapshots (Nachttemperatur, Regen, Wind) | Stündliche Vorhersage für den Stallstandort (DWD oder Open-Meteo, offen). Benötigt: minimale Nachttemperatur, Regenwahrscheinlichkeit/-menge, Wind. Tabelle `weather_snapshots`, Abruf per Cron. Jeder neue Snapshot wird mit der letzten Empfehlung verglichen. |
| 2 | Decken je Pferd: Name, Füllgewicht, Farbe, Lagerort, Foto, Helfernotiz | Tabelle `blankets` (name, fill_g, color, location wie „Haken 3“, photo_url). Das Foto ist wichtig, damit Helfer die richtige Decke greifen. Dazu freier Helfertext je Pferd („Kreuzgurte doppelt schließen“). Bilder in Storage. |
| 2 | Deckenplan: Regeln und Empfehlungslogik | Geordnete Regeln pro Pferd `{ temp_min, temp_max, rain: bool\|null, blanket_id\|null, note }` (`blanket_rules`), vom Besitzer gepflegt. Erste passende Regel für die **Nachtvorhersage** gewinnt: `night_min in [temp_min, temp_max) && (rain == null \|\| rain == will_rain)`. Beispiel Luna: >12 °C keine; 5–12 °C Regendecke nur bei Regen; 0–5 °C 100 g, plus Regendecke bei Regen; <0 °C 200 g. Balu: keine, außer unter −5 °C. Screen `/horses/[id]/blanket-plan`: Hero „Heute Nacht“, Regelliste mit aktiver Markierung, Deckenraster, Helfernotiz. Unit-Tests mit den Beispielen. |
| 2 | Tagesstatus Decken: Eingedeckt, Abgedeckt, Geprüft, mit Historie | Tabelle `blanket_states` je Pferd und Tag: `covered_with`, `changed_at`, `changed_by`. Aktionen „Eingedeckt“, „Abgedeckt“, „Geprüft“ (für Pferde ohne Decke laut Plan, damit sie als erledigt zählen). Historie je Pferd. Realtime. |
| 2 | Screen „Decken heute“ (`/(tabs)/blankets`) | Datum/Nachtzeile, Fortschritt N/7, Hinweis auf 20:30. Offene Pferde zuerst als große Karten (Empfehlung plus Wunsch des Besitzers, Buttons Erledigt/Geprüft), erledigte kompakt. |
| 2 | „Letzte Person“-Erinnerung | Cron zur `stables.reminder_time` (Standard 20:30). Genau 1 Person da: Push an sie mit den offenen Pferden. 0 Personen und offene Pferde: Push an deren Besitzer. Mehr als 1: alle 15 min neu prüfen bis 22:00 oder bis alles erledigt ist. Zusätzlich beim Tippen auf „Ich gehe“ als letzte Person. Pro Nutzer abschaltbar. Legt die Push-Grundlage an (Expo Notifications, Token-Registrierung). |
| 3 | Wetterumschwung-Benachrichtigung | Bei jedem Forecast-Update die Empfehlung neu berechnen und vergleichen. Ändert sie sich für ein Pferd des Nutzers (Besitzer oder RB), nachdem der Tagesstatus gesetzt war, geht ein Push raus. |
| 3 | Start-Screen (`/(tabs)/index`) | Begrüßung, Wetter-Hero (Nachttemperatur und Deckenempfehlung für meine Pferde), Kachel Anwesenheit, Kachel Deckenfortschritt, zwei offene Anfragen (ab M4). |

## M4 Anfragen

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 2 | Anfragen: Datenmodell und Annehmen-Logik | `requests` und `request_assignees`. Felder: type, horse_id, created_by, date/date_range, time_range, location, description, tasks (Checkliste), helpers_needed (Standard 1), status (`open`, `assigned`, `done`, `cancelled`), recurring_rule, remind_at, payload (jsonb). Anfragen gehen an **alle** in der Stallgasse, nicht nur an Anwesende. „Mach ich / Ich komme mit“ fügt einen Helfer hinzu; bei `assignees.length == helpers_needed` wird die Anfrage `assigned`. Push an Ersteller und Helfer. |
| 2 | Screen Anfragen (`/(tabs)/requests`) | Filter-Pills, Karten je Typ (Icon-Kachel, Aufgaben-Chips, Annehmen-Button), „+ Neu“. |
| 2 | Neue Anfrage (`/requests/new`) | Typ-Kacheln, Zeilen für Pferd/Wann/Wo/Helfer, Aufgaben-Chips, Schalter „Mitfahrgelegenheit“, Helfer-Erinnerung, Senden. |
| 2 | Typ `blanket`: Eindecken/Abdecken | Eine Deckenanfrage schließt sich automatisch, wenn der Helfer den Tagesstatus des Pferdes setzt. Optional Rückmeldung mit Foto oder Text („eingedeckt, 200 g“). |
| 3 | Typ `show_helper` („Turniertrottel“) | Turnier, Datum, Prüfungszeiten und Aufgaben: Pferd halten, Abreiten begleiten, Filmen, Startnummer holen, Hänger einladen. Mitfahrt im Hänger-Auto möglich. Details in `payload`. Später zum Turnier-Event ausbaubar (v2). |
| 3 | Typ `ride_share`: Hängerplatz | Hängerplatz zum Turnier oder Lehrgang, Abfahrtszeit, freie Plätze in `payload`. |
| 3 | Typ `exercise` („Bewegen“) | Longieren oder Reiten. Trägt die Trainingsregeln des Pferdes mit, auch den Reha-Plan (ab M6/M7). |
| 3 | Typen `feed_or_turnout`, `appointment_companion`, `other` | Einmalig oder für einen Urlaubszeitraum (Füttern/Raus). Termin begleiten: Pferd zu Hufschmied oder Tierarzt vorführen. |
| 3 | Wiederkehrende Anfragen | `recurring_rule`, z. B. jeden Mittwochabend reinholen. |
| 3 | Helfer-Erinnerung | Standard: am Vortag um 18:00 mit Details und Checkliste. Feld `remind_helper_at`. |
| 3 | Helfer-Kalender und Export | Alle angenommenen Anfragen in einer Liste, Export in den Gerätekalender. |
| 4 | Optionaler „Danke“-Zähler | Leichter Zähler ohne Gamification-Druck. |

## M5 Pferdeakte und Auffälligkeiten

Bereits angelegt: JAN-12 (Fälligkeiten je Pferd mit Erinnerungen).

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 2 | Pferdeakte (`/(tabs)/horses/[id]`) | Kopf mit Name, Geschlecht, Alter, Rasse, Box, Besitzer. Notfallkarte, vier Gesundheitskacheln, Auffälligkeiten, Dokumente. |
| 2 | Notfallkarte (für die ganze Stallgasse sichtbar) | Telefon des Besitzers (Tippen zum Anrufen), Tierarzt-Telefon, Notfallhinweis (z. B. Kolik-Neigung), Notfallmedikation, Dauermedikation (z. B. Equioxx abends, erzeugt Erinnerung), Allergien, Gewicht, Versicherung. Tabellen `emergency_contacts`, Felder an `horses`. |
| 2 | Auffälligkeit melden (`/observations/new`) | Pferd, Kategorie-Chips (Husten, Lahmheit, Verletzung, Frisst nicht, Kolik, Verhalten, Decke/Ausrüstung, Sonstiges), Körperstelle mit einfachem Pferde-Piktogramm (vorne/hinten links/rechts), Beschreibung, Foto/Video, Dringlichkeit `info`, `check`, `urgent`. Benachrichtigt Besitzer und RBs. Tabelle `observations`. |
| 2 | Dringende Auffälligkeit: Notfallkarte und Push an alle Anwesenden | `urgent` öffnet die Notfallkarte und schickt einen Push an alle, die gerade da sind. Ob zusätzlich SMS/Anruf, ist eine offene Entscheidung. |
| 3 | Auffälligkeiten-Liste je Pferd | Status „Beobachten“ und „Erledigt“, Button „Melden“. Kann in einen Reha-Plan umgewandelt werden (M7). |
| 3 | Dokumente je Pferd | Equidenpass, Impfnachweis, Versicherung als Dateien in Storage. |
| 4 | Terminbündelung | Nähert sich ein tierärztlich relevanter Termin, zeigen, wie viele andere Pferde im Gang ebenfalls fällig sind. |

## M6 Training

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 2 | Trainingsprofil (`/horses/[id]/training-profile`, nur Besitzer) | `discipline` (Dressur, Springen, Vielseitigkeit, Freizeit, Western, Jungpferd) plus Level; `allowed_activities` (Halle, Platz, Ausritt, Longe, Springen, Bodenarbeit, Führanlage) je `on`, `off` oder `conditional(note)`; `shows`, Saisonende; Rhythmus (Einheiten pro Woche 4–5, Ruhetage 1–2, max. Dauer 60 min, Pflicht-Ruhetag nach Turnier); `rb_rules` pro RB (erlaubte Aktivitäten, max. Level, allein ausreiten, Turniere); Status `fit`, `reha`, `pause`. |
| 2 | Empfehlungsgeber „Was heute?“ (regelbasiert) | Kandidaten = erlaubte Aktivitäten. Bewertung: Fokus seit ≥2 Tagen nicht trainiert (+), Last der letzten 2 Tage (intensiv → eher leicht), Tage bis Turnier (≤1 nur leicht, 2–3 letzte intensive Einheit ok), Wetter (Regen → Halle), verfügbare Zeit, RB-Regeln. Bei aktiver Reha genau ein Kandidat aus der Phase. Gibt Top 3 mit dem stärksten Grund als Satz zurück. Unit-Tests. |
| 2 | Screen „Was heute?“ (`/(tabs)/training`) | Pferde-Umschalter, Statuskarte mit 7-Tage-Punkten, Kontextzeile, Empfehlung als Hero, zwei Alternativen, Hinweis auf ausgeblendete Aktivitäten („Springen ausgeblendet: laut Profil nicht für Luna“), Buttons Starten, Ablauf, Wählen, „Nur eintragen“ (3 Taps), Profil, Reha-Plan. |
| 3 | Übungsbibliothek | Tabelle `exercises` (Disziplin, Level, Ziel-Tags, Schritte). Füllt „Ablauf“ und schlägt nach „Sitzt“ die nächste Progression vor (z. B. Schulterherein → Travers). |
| 3 | Woche (`/training/week`) | Mo–So-Zeilen: wer, was, Status (erledigt, heute, offen, „Niemand eingetragen“ mit „Ich“-Button, fester Ruhetag). Turniertag mit Prüfungszeiten und Helfer. Lastbalken mit 7 Segmenten und einem Satz zur Einordnung gegen den Rhythmus. Mit RB geteilt; RB darf freie Tage übernehmen, aber keine Regeln ändern. Tabelle `week_slots`. |
| 3 | „Nur eintragen“: schnelles Loggen ohne Tracking | Einheit in drei Taps erfassen. |
| 3 | Abschluss (`/training/session/finish`) | Zusammenfassung (Typ, Dauer, Gangarten- und Handwechsel-Anteile). „Luna war …“: Frisch, Locker, Müde, Klemmig. Fokus-Bewertung: Schwer, Besser, Sitzt. Notiz, Schalter „RB sieht das“, Link „Etwas aufgefallen?“ (Auffälligkeit mit vorbelegtem Pferd). Speichern aktualisiert Woche, Last und Empfehlung. |
| 3 | Lastscore je Einheit | `minutes × intensity(type, canter share)`. Wochenbalken: leicht <30, mittel 30–60, intensiv >60 (später feinjustieren). |

## M7 Tracking und Reha

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 3 | Tracking Ausritt (GPS) | Screen `/training/session`. GPS-Track, Distanz, Ø-Geschwindigkeit, Höhenmeter, Karte nach Gangart eingefärbt (`react-native-maps`), Pause/Beenden. Hintergrund-Standort mit `expo-location`. Optional später Pulsdaten von der Smartwatch. |
| 3 | Gangarterkennung | `expo-sensors` Accelerometer mit 50 Hz, Fenster-FFT/Peak-Erkennung. Schritt ≈ 0,8–1,2 Hz, Trab ≈ 1,3–1,8 Hz (Zweitakt, kräftig vertikal), Galopp ≈ 1,6–2,2 Hz mit asymmetrischem Muster. Draußen mit GPS-Geschwindigkeit fusionieren (Schritt <7 km/h, Trab 7–16, Galopp >16). Rohfenster speichern für spätere Modellverbesserung. Kalibrierung pro Nutzer (Handy in Tasche oder am Arm). |
| 3 | Tracking Halle/Platz | GPS aus. Gangart per Accelerometer, große Buttons Schritt/Trab/Galopp nur zum Korrigieren, Button „Handwechsel“ (Zeit je Rein), Übungs-Checkliste aus „Ablauf“, Zieldauer. |
| 3 | Reha-Plan: Datenmodell und Screen (`/horses/[id]/reha`) | Tabelle `reha_plans` (Diagnose, Tierarzt, Start, Phasen, Kontrolltermin, aktiv, Verweis auf Auffälligkeit). Phase: Name, Dauer in Tagen, erlaubte Aktivität, Dauer von–bis in Minuten, Bedingungen (Boden, kein Trab …), Abbruchkriterien. Beispiel: Boxenruhe 5 d → Schritt führen 10→20 min 9 d → Schritt reiten 20→40 min 14 d → Trab aufbauen 2→15 min 14 d. Karte „Heute erlaubt“, Button „Heute erledigt“, Tierarztkontrolle mit Erinnerung. |
| 3 | Reha-Integration in Training und Anfragen | Solange aktiv: „Was heute?“ zeigt nur die erlaubte Einheit, die Woche zeigt Reha-Einträge, `exercise`-Anfragen tragen die Regel, Besitzer und RB sehen den Plan, Profilstatus = `reha`. |
| 4 | Reha-Plan aus Auffälligkeit erzeugen | Direkt aus einer gemeldeten Auffälligkeit oder manuell anlegen; „Auffälligkeit melden“ auch aus dem Reha-Screen. |

## M8 Erinnerungen und Einstellungen

Bereits angelegt: JAN-18 (Push), JAN-19 (Datenschutz), JAN-20 (Pilot und Store).

| Prio | Titel | Beschreibung |
| --- | --- | --- |
| 3 | Erinnerungszentrale (`/reminders`) | Hero „Deckencheck“, Liste gruppiert nach „Heute“ und „Diese Woche“. Quellen: Letzte-Person-Check (20:30), Wetterumschwung, Dauermedikation, Helfer-Erinnerungen, Trainingsplan am Vorabend (optional), Impf-/Hufschmied-/Tierarzttermine, Reha-Tierarztkontrolle, neue Anfragen (Opt-in). Tabelle `reminders`. |
| 3 | Einstellungen: Schalter je Erinnerungstyp | Pro Typ ein-/ausschaltbar, Erinnerungszeit der Stallgasse, Sichtbarkeit der Anwesenheit. |
