-- M6 Training (JAN-91): the global exercise library (stable_id NULL) for every installation,
-- not only the demo seed. Every row names the pages it is based on; nothing is made up, a step
-- without a figure in the sources leaves it out. Main sources: FN member magazine "Pferd und
-- Mensch" (FN_MAG = https://magazin.pferdesport-deutschland.de/project/), FN Merkblätter for the
-- badges Bodenarbeit, Vormustern and Longieren, de.wikipedia and established riding magazines.
-- Levels: beginner = basic training and Klasse E/A, intermediate = A* to L*, advanced = L** and M
-- (groundwork and lunging: the FN badge stages). Existing rows with these IDs (earlier demo seed)
-- are replaced; later corrections need a new migration.

INSERT INTO exercises (id, stable_id, discipline, level, goal_tags, title, steps) VALUES

-- --- dressage ---
-- FN_MAG/lektion-im-fokus-09-2021/
-- https://www.wehorse.com/de/blog/zuegel-aus-der-hand-kauen-lassen/
('00000000-0000-4000-8000-000000000801', NULL, 'dressage', 'beginner', '{losgelassenheit,anlehnung,takt}', 'Zügel-aus-der-Hand-kauen-lassen',
   '[
     "Im Arbeitstrab auf der Zirkellinie reiten, wenn das Pferd gelöst an den Hilfen steht.",
     "Die Hände allmählich vorgehen lassen, die Schenkel treiben weiter: Das Pferd dehnt sich vorwärts-abwärts, das Maul mindestens bis auf Höhe des Buggelenks.",
     "Die Verbindung zum Pferdemaul bleibt erhalten, Takt und Tempo bleiben gleich.",
     "Nach einer kurzen Reprise die Zügel wieder aufnehmen, damit das Pferd nicht auf die Vorhand kommt."
   ]'),
-- https://www.pferderevue.at/content/pferderevue/pferderevue/de/magazin/ausbildung/2016/09/gymnastizierung_purmitschlangenlinieund-tour.html
-- https://de.wikipedia.org/wiki/Hufschlagfigur
-- FN_MAG/hufschlagfiguren-auf-der-richtigen-linie-02-2023
('00000000-0000-4000-8000-000000000802', NULL, 'dressage', 'beginner', '{biegung,losgelassenheit,geraderichten}', 'Schlangenlinien durch die ganze Bahn',
   '[
     "An der kurzen Seite beginnen; im 20 × 40-m-Viereck meist drei oder vier Bögen, dazwischen geradeaus parallel zur kurzen Seite.",
     "In jedem Bogen stellt der innere Zügel, der äußere gibt etwas nach und begrenzt; der innere Schenkel am Gurt biegt, der äußere hält die Hinterhand in der Spur.",
     "Beim Überqueren der Mittellinie das Pferd geraderichten und umstellen wie bei einem Handwechsel.",
     "Auf gleichmäßigen Takt und gleich große Bögen achten, nicht am inneren Zügel ziehen."
   ]'),
-- https://de.wikipedia.org/wiki/Vorhandwendung
-- FN_MAG/diagonale-hilfengebung-verstehen-09-2023/
('00000000-0000-4000-8000-000000000803', NULL, 'dressage', 'beginner', '{durchlaessigkeit,losgelassenheit}', 'Vorhandwendung',
   '[
     "Auf dem zweiten Hufschlag geschlossen halten, damit Kopf und Hals zur Bande Platz haben.",
     "Das Pferd in Richtung der Wendung stellen, nicht biegen; der neue innere Schenkel liegt seitwärtstreibend hinter dem Gurt.",
     "Mit kleinen Impulsen tritt die Hinterhand Schritt für Schritt im Kreisbogen um die Vorhand; äußerer Schenkel und äußerer Zügel verwahren.",
     "Bei Unsicherheit zwischendurch kurz halten, zum Abschluss wieder geschlossen halten."
   ]'),
-- FN_MAG/10-tipps-08-2022/
-- https://de.wikipedia.org/wiki/Schenkelweichen
('00000000-0000-4000-8000-000000000804', NULL, 'dressage', 'beginner', '{losgelassenheit,durchlaessigkeit,takt}', 'Schenkelweichen',
   '[
     "Im Mittelschritt oder Arbeitstrab entlang der Bande oder auf einer Diagonalen, mit Stellung, aber ohne Längsbiegung.",
     "Der innere Schenkel treibt knapp hinter dem Gurt vorwärts-seitwärts, der äußere verwahrt; der innere Zügel stellt, der äußere begrenzt die Halsstellung.",
     "Die inneren Beine treten vor und über die äußeren; die Abstellung bleibt unter 45 Grad, das Vorwärts bleibt erhalten.",
     "Fließend zwischen Schenkelweichen und Geradeausreiten wechseln."
   ]'),
-- FN_MAG/lektion-im-fokus-07-2023/
-- FN_MAG/lektion-im-fokus-03-2022/
('00000000-0000-4000-8000-000000000805', NULL, 'dressage', 'beginner', '{biegung,balance,durchlaessigkeit}', 'Volte',
   '[
     "Vor der Volte mit einigen halben Paraden Takt, Gleichgewicht und Selbsthaltung sichern.",
     "Gewicht innen, der innere Schenkel für Längsbiegung und fleißiges Abfußen, der äußere verhindert das Ausweichen der Hinterhand; der äußere Zügel hält die Linie.",
     "Beim Abwenden zum Scheitelpunkt der Volte schauen, von dort zurück zum Ausgangspunkt.",
     "Zunächst größere Volten (10 m) reiten; ab Klasse L wird die 8-m-Volte verlangt."
   ]'),
-- FN_MAG/lektion-im-fokus-11-12-2021/
('00000000-0000-4000-8000-000000000806', NULL, 'dressage', 'beginner', '{durchlaessigkeit,balance}', 'Rückwärtsrichten',
   '[
     "Aus dem geschlossenen Halten auf gerader Linie beginnen.",
     "Beidseitig belastende Gewichtshilfe und vorwärtstreibende Schenkel; die annehmende oder aushaltende Zügelhilfe lenkt die Bewegung nach rückwärts.",
     "Das Pferd tritt im diagonalen Beinpaar rückwärts, in der Grundausbildung etwa eine Pferdelänge (drei bis vier Tritte).",
     "Zu starke Zügeleinwirkung vermeiden, sonst schleppt das Pferd die Füße."
   ]'),
-- FN_MAG/lektion-im-fokus-01-2022/
('00000000-0000-4000-8000-000000000807', NULL, 'dressage', 'intermediate', '{versammlung,durchlaessigkeit,biegung}', 'Kurzkehrt',
   '[
     "Aus dem Mittelschritt mit einigen halben Paraden aufnehmen, aus dem Trab ein bis zwei Schritte vorher zum Schritt durchparieren, ohne zu halten.",
     "Das Pferd ist in Bewegungsrichtung gestellt und gebogen, die Vorhand beschreibt einen Halbkreis um die Hinterhand; das innere Hinterbein tritt fast auf der Stelle, der Viertakt bleibt klar.",
     "Vermehrt innen sitzen: Der innere Zügel leitet ein, der innere Schenkel erhält Fleiß und Biegung, der äußere Schenkel begrenzt, ohne herumzudrücken.",
     "Anfangs nur um 45 oder 90 Grad wenden, danach fleißig geradeaus reiten."
   ]'),
-- https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-uebergaenge.html
-- FN_MAG/10-tipps-11-12-2019/
-- FN_MAG/lektion-im-fokus-07-2022/
('00000000-0000-4000-8000-000000000808', NULL, 'dressage', 'beginner', '{durchlaessigkeit,losgelassenheit}', 'Übergänge',
   '[
     "Schritt-Trab-Schritt an markierten Punkten reiten.",
     "Vor jedem Übergang eine halbe Parade geben; beim Durchparieren erst ausatmen, dann Gewicht, dann Schenkel und zuletzt Zügel.",
     "Nach dem Übergang den Takt sofort wiederfinden.",
     "Steigern: Trab-Galopp-Trab und Übergänge innerhalb der Gangart."
   ]'),
-- FN_MAG/reiten-von-hufschlagfiguren-11-12-2023/
-- FN_MAG/hufschlagfiguren-auf-der-richtigen-linie-02-2023
-- https://www.tipps-zum-pferd.de/hufschlagfigur-zirkel-verkleinern-vergroessern_tipp_328.html
('00000000-0000-4000-8000-000000000809', NULL, 'dressage', 'beginner', '{biegung,losgelassenheit}', 'Zirkel verkleinern und vergrößern',
   '[
     "Auf dem Zirkel im Arbeitstrab anreiten, Takt und Tempo halten.",
     "Spiralförmig verkleinern, pro Runde etwa eine Pferdebreite: Gewicht innen, der äußere Zügel und der verwahrende äußere Schenkel führen das Pferd nach innen.",
     "Je kleiner der Kreis, desto mehr das Tempo verkürzen; den nächstkleineren Kreis erst reiten, wenn der vorige gelingt.",
     "Mit dem vorwärts-seitwärts treibenden inneren Schenkel an den äußeren Zügel wieder vergrößern."
   ]'),
-- FN_MAG/lektion-im-fokus-03-2022/
-- https://de.wikipedia.org/wiki/Schulterherein
('00000000-0000-4000-8000-000000000810', NULL, 'dressage', 'advanced', '{seitengaenge,biegung,versammlung,geraderichten}', 'Schulterherein',
   '[
     "Aus der Ecke heraus oder aus einer Volte beginnen und die Längsbiegung aus der Wendung mitnehmen.",
     "Die Vorhand etwa 30 Grad in die Bahn führen, die Hinterhand bleibt auf dem Hufschlag; das Pferd geht auf drei Hufschlaglinien, gleichmäßig um den inneren Schenkel gebogen.",
     "Gewicht innen, der innere Schenkel treibt und erhält die Biegung, der äußere begrenzt die Hinterhand; der äußere Zügel lässt die Stellung zu.",
     "Takt und Schwung haben Vorrang: nur eine halbe lange Seite reiten, dann geradeaus und loben."
   ]'),
-- FN_MAG/lektion-im-fokus-05-2022/
-- https://de.wikipedia.org/wiki/Travers_(Reitsport)
('00000000-0000-4000-8000-000000000811', NULL, 'dressage', 'advanced', '{seitengaenge,biegung,versammlung}', 'Travers',
   '[
     "An der langen Seite oder, bei jungen Pferden, auf dem Zirkel mit Stellung und Biegung in Bewegungsrichtung beginnen.",
     "Die Vorhand bleibt auf dem Hufschlag, die Hinterhand kommt etwa 30 bis 35 Grad herein; das Pferd geht auf vier Hufschlaglinien.",
     "Gewicht in Bewegungsrichtung, der innere Schenkel für die Biegung, der äußere bringt die Hinterhand herein; die innere Hand nicht über den Mähnenkamm drücken.",
     "Zum Beenden die Hinterhand zurück auf den Hufschlag führen und auf beiden Händen im Wechsel üben."
   ]'),
-- FN_MAG/lektion-im-fokus-10-2022/
-- https://de.wikipedia.org/wiki/Traversale
-- https://www.ehorses.de/magazin/traversale-reiten/
('00000000-0000-4000-8000-000000000812', NULL, 'dressage', 'advanced', '{seitengaenge,biegung,versammlung,durchlaessigkeit}', 'Traversale',
   '[
     "Nur im versammelten Trab oder Galopp reiten; mit flachen, langen Traversalen beginnen, etwa einer halben Traversale von der Mittellinie zum Hufschlag.",
     "Das Pferd ist in Bewegungsrichtung gestellt und gebogen und geht fast parallel zur langen Seite, die Vorhand führt leicht.",
     "Gewicht innen, der innere Schenkel am Gurt, der äußere hinter dem Gurt treibt vorwärts-seitwärts; der innere Zügel stellt, der äußere begrenzt.",
     "Nicht mit dem äußeren Schenkel hinüberschieben, die Hinterhand darf nicht vorausgehen; zum Beenden geradeaus reiten."
   ]'),
-- FN_MAG/lektion-im-fokus-10-2021/
-- FN_MAG/hufschlagfiguren-auf-der-richtigen-linie-02-2023
('00000000-0000-4000-8000-000000000813', NULL, 'dressage', 'intermediate', '{durchlaessigkeit,takt,losgelassenheit}', 'Viereck verkleinern und vergrößern',
   '[
     "Am ersten Wechselpunkt der langen Seite im Mittelschritt oder Arbeitstrab beginnen.",
     "Mit Stellung, ohne Biegung vorwärts-seitwärts parallel zum Hufschlag bis zur Viertellinie; der innere Schenkel knapp hinter dem Gurt, der äußere verwahrt, der äußere Zügel verhindert das Ausfallen über die Schulter.",
     "Eine Pferdelänge geradeaus, dann im Schenkelweichen zurück zum zweiten Wechselpunkt.",
     "Der Takt steht im Vordergrund; zu viel Halsstellung und zu steiles Seitwärts vermeiden."
   ]'),
-- FN_MAG/lektion-im-fokus-07-2022/
('00000000-0000-4000-8000-000000000814', NULL, 'dressage', 'intermediate', '{schwung,takt,durchlaessigkeit}', 'Mitteltrab',
   '[
     "Aus einem gelösten, fleißigen Arbeitstrab zulegen; die Hände geben so weit vor, dass das Pferd seinen Rahmen verlängern kann.",
     "Der Raumgriff wird größer, ohne dass die Tritte eiliger werden; die Hinterhufe treten über die Spur der Vorderhufe.",
     "Der Hals bleibt der höchste Punkt, die Stirn-Nasen-Linie deutlich vor der Senkrechten.",
     "Mit halben Paraden zum Arbeitstrab zurückführen, das fleißige Abfußen bleibt erhalten."
   ]'),
-- FN_MAG/lektion-im-fokus-02-2022/
-- FN_MAG/10-tipps-05-2023/
('00000000-0000-4000-8000-000000000815', NULL, 'dressage', 'intermediate', '{durchlaessigkeit,versammlung,balance}', 'Einfacher Galoppwechsel',
   '[
     "Die Übergänge Galopp-Schritt und Schritt-Galopp zunächst einzeln üben, anfangs mit einer längeren Schrittphase.",
     "Mit halben Paraden aus aktivem Galopp geschmeidig zum Mittelschritt durchparieren und drei bis fünf klare Schritte reiten.",
     "Dann unmittelbar und bergauf im neuen Galopp angaloppieren.",
     "Geeignete Linien: Zirkel verkleinern, halbe Volte, später durch die Länge der Bahn wechseln mit Wechsel über X."
   ]'),
-- FN_MAG/10-tipps-08-2023/
('00000000-0000-4000-8000-000000000816', NULL, 'dressage', 'intermediate', '{versammlung,geraderichten,balance}', 'Außengalopp',
   '[
     "Voraussetzung ist ein ausbalancierter, geradegerichteter Handgalopp mit sicheren Übergängen.",
     "Zum Einstieg im Galopp eine einfache Schlangenlinie an der langen Seite reiten; das Pferd bleibt dabei im bisherigen Galopp.",
     "Innen bleibt die Seite, zu der das Pferd gestellt und gebogen ist; der innere Schenkel treibt an den äußeren Zügel.",
     "Später auf größeren Linien üben; zu viel Stellung und ein blockierender innerer Zügel stören."
   ]'),
-- FN_MAG/lektion-im-fokus-05-2022/
-- https://de.wikipedia.org/wiki/Traversale
('00000000-0000-4000-8000-000000000817', NULL, 'dressage', 'advanced', '{seitengaenge,biegung,versammlung}', 'Renvers',
   '[
     "Über eine einfache Schlangenlinie einsteigen: Sobald die Biegung entsteht, die Renvers-Hilfen geben.",
     "Die Hinterhand bleibt auf dem Hufschlag, die Vorhand wird in die Bahn geführt; das Pferd ist in Bewegungsrichtung gestellt und gebogen und geht auf vier Hufschlaglinien.",
     "Gewicht in Bewegungsrichtung, der innere Schenkel für die Biegung, der äußere führt die Hinterhand; der innere Zügel stellt, der äußere begrenzt.",
     "Zum Beenden die Vorhand zurück auf den Hufschlag führen; die Biegung nicht durch Halsstellung ersetzen."
   ]'),

-- --- jumping (distances for horses) ---
-- FN_MAG/10-tipps-03-2023/
-- FN_MAG/ausbildung-cavaletti-08-2024/
-- https://de.wikipedia.org/wiki/Cavaletti_(Reitsport)
('00000000-0000-4000-8000-000000000820', NULL, 'jumping', 'beginner', '{takt,losgelassenheit}', 'Schrittstangen',
   '[
     "Sechs bis acht Stangen im Abstand von ca. 0,80 m (Großpferd 0,80–0,90 m) auf gerader Linie auslegen und mit dem Maßband nachmessen.",
     "Im gleichmäßigen Schritt mittig über die Stangen reiten.",
     "Die Stangen regeln Takt und Gleichmaß, besonders bei eiligem Schritt."
   ]'),
-- FN_MAG/10-tipps-03-2023/
-- FN_MAG/ausbildung-cavaletti-08-2024/
-- https://reiter-pferde.de/das-geht-locker-loesungsarbeit-mit-stangen-und-cavaletti/
('00000000-0000-4000-8000-000000000821', NULL, 'jumping', 'beginner', '{takt,rhythmus,losgelassenheit}', 'Trabstangen',
   '[
     "Vier Stangen oder Cavaletti im Abstand von ca. 1,30 m (je nach Pferd 1,20–1,40 m) an der langen Seite auslegen.",
     "Mittig anreiten und leichttraben oder im leichten Sitz mitgehen.",
     "Das Pferd gleichmäßig, schwungvoll und im Rhythmus weitertraben lassen.",
     "Später auf sechs Stangen erweitern und auf gebogener Linie reiten."
   ]'),
-- FN_MAG/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017
-- FN_MAG/einstieg-in-das-springen-06-2021/
-- FN_MAG/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017
('00000000-0000-4000-8000-000000000822', NULL, 'jumping', 'intermediate', '{springgymnastik,rhythmus,balance}', 'Gymnastikreihe',
   '[
     "Aufbau: Cavaletti, 2,20 m bis zum Kreuz, 3,40 m bis zum zweiten Kreuz (Einsprung-Aussprung), 6,80 m bis zum Steilsprung, 10,00 m bis zum Oxer.",
     "Zunächst nur die ersten Elemente sehr niedrig aufbauen und die Reihe schrittweise erweitern.",
     "Aus dem Trab anreiten, im leichten Sitz mitgehen und mit der Hand der Kopf-Hals-Bewegung folgen.",
     "Die Abstände sind Zirka-Werte für Großpferde; je nach Pferdegröße und Hindernishöhe anpassen, für Ponys verkürzen."
   ]'),
-- FN_MAG/ausbildung-cavaletti-08-2024/
-- https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-cavaletti.html
('00000000-0000-4000-8000-000000000823', NULL, 'jumping', 'intermediate', '{rhythmus,balance,losgelassenheit}', 'Cavaletti auf dem Zirkel',
   '[
     "An beiden geschlossenen Zirkelseiten je zwei niedrige Cavaletti fächerförmig auslegen, in der Mitte 1,50 m Abstand; den ersten Hufschlag frei lassen.",
     "Leichttrabend auf dem 20-m-Zirkel über die Cavaletti traben, dann durch den Zirkel wechseln.",
     "Später vier Cavaletti im Fächer: Pferde mit weniger Raumgriff traben weiter innen, Pferde mit mehr Raumgriff weiter außen."
   ]'),
-- FN_MAG/10-tipps-03-2023/
-- FN_MAG/ausbildung-cavaletti-08-2024/
-- https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-cavaletti.html
('00000000-0000-4000-8000-000000000824', NULL, 'jumping', 'intermediate', '{rhythmus,balance,takt}', 'Galoppcavaletti',
   '[
     "Zwei oder mehr Cavaletti im Abstand von ca. 3 m aufstellen.",
     "Im ruhigen Arbeitsgalopp im leichten Sitz mittig anreiten.",
     "Im Fächer auf der Zirkellinie (innen ca. 2 m, außen ca. 3 m) die Galoppsprünge variieren: innen verkürzen, außen verlängern.",
     "Die höchste Cavaletti-Stufe nur im Galopp verwenden."
   ]'),
-- FN_MAG/einstieg-in-das-springen-06-2021/
-- FN_MAG/spruenge-passend-anreiten-05-2022
('00000000-0000-4000-8000-000000000825', NULL, 'jumping', 'intermediate', '{rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Distanzen reiten',
   '[
     "Zwei Cavaletti auf gerader Linie im Abstand von 22 m aufstellen.",
     "Im Arbeitsgalopp mit sechs Galoppsprüngen dazwischen reiten und die Sprünge laut mitzählen.",
     "Klappen sechs Sprünge gleichmäßig, den Galopp auf sieben Sprünge verkürzen, danach auf fünf verlängern.",
     "Den Galopp schon vor dem ersten Cavaletti anpassen, danach nur den Rhythmus halten."
   ]'),
-- FN_MAG/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017
-- FN_MAG/einstieg-in-das-springen-06-2021/
('00000000-0000-4000-8000-000000000826', NULL, 'jumping', 'beginner', '{springgymnastik,balance}', 'Kreuzsprung mit Vorlegestange',
   '[
     "Ein Kreuz aufbauen und die Vorlegestange ca. 2,20–2,50 m davor parallel auslegen; bei unerfahrenen Pferden Fangständer verwenden.",
     "Im gleichmäßigen Trab mittig anreiten und im leichten Sitz mitgehen.",
     "Die Linie vor und nach dem Sprung einhalten; Pylonen zeigen den Weg.",
     "Aus dem Galopp liegt die Vorlegestange ca. 3 m vor dem Sprung."
   ]'),
-- FN_MAG/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017
('00000000-0000-4000-8000-000000000827', NULL, 'jumping', 'advanced', '{springgymnastik,rhythmus,aufmerksamkeit}', 'Oxerreihe',
   '[
     "Vier Hoch-Weit-Sprünge in einer Reihe aufbauen: 6,50–7,00 m für einen Galoppsprung, 9,70–10,20 m für zwei.",
     "Aus ruhigem Tempo mittig anreiten und gerade springen; das Pferd soll energisch abspringen, nicht aus höherem Tempo.",
     "Springreihen nur dosiert einsetzen, Einsprung-Aussprung-Reihen kosten viel Kraft."
   ]'),

-- --- groundwork ---
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000830', NULL, 'groundwork', 'beginner', '{vertrauen,aufmerksamkeit}', 'Führen und Halten',
   '[
     "Links vom Pferd zwischen Pferdekopf und Schulter gehen, den Strick in der rechten Hand bis zu 50 cm unterhalb des Halfters.",
     "Gleichzeitig antreten; das Pferd richtet sich nach dem Tempo des Führenden.",
     "Das Anhalten über die Körpersprache einleiten, bei Bedarf mit Stimme oder leichten Impulsen am Strick.",
     "Punktgenau halten, möglichst ohne Handeinwirkung und mit durchhängendem Strick."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- FN_MAG/10-tipps-05-2020
-- https://www.cavallo.de/reittraining/das-bringt-rueckwaertsrichten-dem-pferd/
('00000000-0000-4000-8000-000000000831', NULL, 'groundwork', 'intermediate', '{aufmerksamkeit,balance}', 'Rückwärtsrichten an der Hand',
   '[
     "Aus dem Halten mit der Stimme ankündigen („Zurück“).",
     "Mit einem Impuls am Halfter auffordern, bei Bedarf mit leichtem Druck der Hand am Buggelenk unterstützen.",
     "Sofort nachgeben, sobald das Pferd zurücktritt; schon die Tendenz nach hinten belohnen.",
     "Eine Pferdelänge gerade und im Zweitakt rückwärts, dann wieder halten."
   ]'),
-- https://www.cavallo.de/reittraining/freiarbeit-vertieft-die-freundschaft-zum-pferd/
('00000000-0000-4000-8000-000000000832', NULL, 'groundwork', 'advanced', '{vertrauen,aufmerksamkeit}', 'Freiarbeit',
   '[
     "In einer umzäunten Umgebung wie Reitplatz oder Halle arbeiten.",
     "Am Strick beginnen und die Hilfen über Stimme und Körpersprache aufbauen, Strick und Gerte unterstützen.",
     "Reagiert das Pferd sicher, den Strick abnehmen und die Hilfsmittel nach und nach weglassen."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000833', NULL, 'groundwork', 'beginner', '{vertrauen,aufmerksamkeit}', 'Anbinden und Stillstehen',
   '[
     "Den Anbindestrick mit Panikhaken am Halfter einhaken.",
     "Mit Sicherheitsknoten anbinden, ca. 60–80 cm zwischen Panikhaken und Anbindering.",
     "So viel Spielraum lassen, dass das Pferd sich bewegen, aber nicht in den Strick treten kann.",
     "Das Pferd steht ruhig auf allen vier Beinen, bis ein neues Signal kommt."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.loesdau.de/specials/loesdau-lessons/bodenarbeit-mit-sabine-ellinger/teil-3-die-zweite-grunduebung-der-bodenarbeit/
('00000000-0000-4000-8000-000000000834', NULL, 'groundwork', 'beginner', '{durchlaessigkeit,aufmerksamkeit}', 'Vorhand weichen lassen',
   '[
     "Den Pferdekopf halten und leicht entgegen der Bewegungsrichtung stellen.",
     "Mit auffordernder Körpersprache und Stimme um die Vorhand herumtreten lassen, bei Bedarf hinter der Gurtlage berühren.",
     "Das innere Hinterbein kreuzt vor dem äußeren.",
     "Sofort nachgeben, sobald das Pferd weicht."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/
('00000000-0000-4000-8000-000000000835', NULL, 'groundwork', 'intermediate', '{durchlaessigkeit,balance}', 'Seitwärts weichen an der Hand',
   '[
     "An der Bande oder auf einer vorgegebenen Linie beginnen.",
     "In Schulterhöhe stehen und erst die Schulter, dann die Hinterhand weichen lassen.",
     "Auf gleichmäßiges Übertreten achten.",
     "Das Pferd weicht willig und flüssig; Körpersprache, Berührung und Stimme dosiert einsetzen."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000836', NULL, 'groundwork', 'advanced', '{durchlaessigkeit,balance,biegung}', 'Wendung um die Hinterhand an der Hand',
   '[
     "Der Pferdekopf bleibt beim Führenden, die Vorhand wird um die Hinterhand herumgeführt.",
     "Das äußere Vorderbein kreuzt über das innere.",
     "Die Hinterbeine treten weiter mit und bleiben nicht stehen.",
     "Mit 90 Grad beginnen und auf 180 oder 360 Grad steigern."
   ]'),
-- https://www.renate-gassner.de/uploads/media/Merkblatt_Vormustern_.pdf
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000837', NULL, 'groundwork', 'intermediate', '{takt,aufmerksamkeit}', 'Vorführen auf der Dreiecksbahn',
   '[
     "Das Pferd 3–4 m vor dem Betrachter offen aufstellen, sodass alle vier Beine zu sehen sind.",
     "Auf Trense führen: beide Zügel 3–4 Handbreit hinter den Gebissringen in der rechten Hand, leicht durchhängend; der Führende geht links.",
     "Die erste Seite im Schritt führen, nach der Wendung antraben und vor der nächsten Wendung zum Schritt durchparieren; immer nach rechts wenden.",
     "Zum Schluss vorbeiführen, rechts wenden und von der anderen Seite offen aufstellen."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
('00000000-0000-4000-8000-000000000838', NULL, 'groundwork', 'intermediate', '{aufmerksamkeit,balance,vertrauen}', 'Stangenarbeit an der Hand',
   '[
     "Etwa 3 m vor der Stange das Tempo verlangsamen.",
     "So anhalten, dass die Vorhand vor und die Hinterhand hinter der Stange steht.",
     "Im Stangenlabyrinth im Schritt führen, ohne Stangen zu berühren oder seitlich auszubrechen.",
     "Bei Schwierigkeiten die Aufgabe leichter machen und mit einer gelungenen Ausführung enden."
   ]'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
('00000000-0000-4000-8000-000000000839', NULL, 'groundwork', 'intermediate', '{vertrauen,gelassenheit}', 'Über die Plane führen',
   '[
     "Zuerst in der Halle oder auf einem umzäunten Platz üben.",
     "Der Führende geht zwischen Plane und Pferd und hält dem Pferd den Fluchtweg vorwärts-seitwärts frei.",
     "Den Reiz nur so weit steigern, dass das Pferd nicht flüchtet; Neugier und Untersuchen zulassen.",
     "Selbst ruhig bleiben und jeden Schritt zur Plane mit einer Pause belohnen."
   ]'),

-- --- lunging ---
-- FN_MAG/longieren-aber-sinnvoll-08-2019/
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
('00000000-0000-4000-8000-000000000840', NULL, 'lunge', 'beginner', '{losgelassenheit,takt,vertrauen}', 'Aufwärmen im Schritt',
   '[
     "Mit Handschuhen und festem Schuhwerk, ohne Sporen longieren; die Longe hat Handschlaufe und Karabiner, keinen Wirbelkarabiner.",
     "Auf einem umzäunten Zirkel mit mindestens 16 m Durchmesser und rutschfestem Boden arbeiten.",
     "Longe, Peitsche und Pferd bilden ein Dreieck; das Pferd Runde für Runde auf die große Zirkellinie hinausgehen lassen.",
     "10–15 Minuten unausgebunden im Schritt aufwärmen; die ganze Einheit dauert höchstens 35–40 Minuten."
   ]'),
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- FN_MAG/longieren-aber-sinnvoll-08-2019/
('00000000-0000-4000-8000-000000000841', NULL, 'lunge', 'beginner', '{durchlaessigkeit,aufmerksamkeit,takt}', 'Übergänge an der Longe',
   '[
     "Übergänge Schritt–Trab–Galopp und zurück bis zum Halten longieren.",
     "Eine tiefe, ruhige Stimme zum Zurückholen, eine höhere für mehr Aufmerksamkeit; es zählt der Tonfall, nicht die Lautstärke.",
     "Zum Durchparieren Paraden an der Longe mit einem klaren Kommando verbinden.",
     "Eine neue Gangart erst verlangen, wenn das Pferd in der aktuellen ausbalanciert ist."
   ]'),
-- FN_MAG/longieren-aber-sinnvoll-08-2019/
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://www.cavallo.de/reittraining/gymnastik-an-der-longe/
('00000000-0000-4000-8000-000000000842', NULL, 'lunge', 'beginner', '{aufmerksamkeit,durchlaessigkeit}', 'Handwechsel an der Longe',
   '[
     "Das Pferd mit der Stimme aufmerksam machen und in ruhiger Haltung nach innen holen, die Longe bleibt leicht gespannt.",
     "Longe und Peitsche die Hand wechseln lassen und die Ausbindezügel für die neue Hand anpassen.",
     "Auf der neuen Hand wieder hinauslongieren.",
     "Häufig die Hand wechseln, das ist an der Longe besonders wichtig."
   ]'),
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
('00000000-0000-4000-8000-000000000843', NULL, 'lunge', 'intermediate', '{biegung,balance,durchlaessigkeit}', 'Zirkel verkleinern, vergrößern und verlagern',
   '[
     "Im Arbeitstrab den Zirkel allmählich verkleinern und wieder vergrößern.",
     "Takt, Losgelassenheit und Anlehnung dabei erhalten.",
     "Den Zirkel verlagern und dabei die Trabtritte zweimal verlängern und verkürzen.",
     "Im Galopp die Galoppsprünge zweimal verlängern und verkürzen."
   ]'),
-- FN_MAG/ausbildung-cavaletti-08-2024/
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
('00000000-0000-4000-8000-000000000844', NULL, 'lunge', 'intermediate', '{takt,balance,losgelassenheit}', 'Longieren über Trabstangen',
   '[
     "Mit einer Stange beginnen; die nächste erst dazulegen, wenn das Pferd die erste gelassen überwindet.",
     "Die Abstände mit dem Maßband messen: Schritt ca. 0,80–0,90 m, Trab ca. 1,20–1,40 m.",
     "Vier Stangen sind ideal, bei fünf oder sechs ist Schluss.",
     "Stangenarbeit regelmäßig, aber höchstens alle zwei Tage einplanen."
   ]'),
-- FN_MAG/longieren-aber-sinnvoll-08-2019/
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
('00000000-0000-4000-8000-000000000845', NULL, 'lunge', 'intermediate', '{anlehnung,losgelassenheit,takt}', 'Longieren mit Ausbindezügeln',
   '[
     "Die Ausbindezügel erst nach dem unausgebundenen Aufwärmen im Schritt einschnallen.",
     "Länge prüfen: vor dem Pferd stehen, das Gebiss auf beiden Seiten fassen und den Kopf vorsichtig heranführen, bis beide Zügel gleichmäßig leicht anstehen; die Stirn-Nasen-Linie bleibt vor der Senkrechten.",
     "Bei jedem Handwechsel anpassen; den Trensenzügel nie zum Ausbinden verwenden.",
     "Zum Schluss etwa 10 Minuten unausgebunden im Schritt entspannen lassen."
   ]')
ON CONFLICT (id) DO UPDATE SET
    discipline = EXCLUDED.discipline, level = EXCLUDED.level, goal_tags = EXCLUDED.goal_tags,
    title = EXCLUDED.title, steps = EXCLUDED.steps
WHERE exercises.stable_id IS NULL;

-- Progressions (follow-up exercise), set once all rows exist.
UPDATE exercises e SET next_exercise_id = p.next
FROM (VALUES
    ('00000000-0000-4000-8000-000000000801'::uuid, '00000000-0000-4000-8000-000000000814'::uuid),
    ('00000000-0000-4000-8000-000000000802'::uuid, '00000000-0000-4000-8000-000000000816'::uuid),
    ('00000000-0000-4000-8000-000000000803'::uuid, '00000000-0000-4000-8000-000000000804'::uuid),
    ('00000000-0000-4000-8000-000000000804'::uuid, '00000000-0000-4000-8000-000000000813'::uuid),
    ('00000000-0000-4000-8000-000000000805'::uuid, '00000000-0000-4000-8000-000000000810'::uuid),
    ('00000000-0000-4000-8000-000000000806'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000807'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000808'::uuid, '00000000-0000-4000-8000-000000000814'::uuid),
    ('00000000-0000-4000-8000-000000000809'::uuid, '00000000-0000-4000-8000-000000000815'::uuid),
    ('00000000-0000-4000-8000-000000000810'::uuid, '00000000-0000-4000-8000-000000000811'::uuid),
    ('00000000-0000-4000-8000-000000000811'::uuid, '00000000-0000-4000-8000-000000000817'::uuid),
    ('00000000-0000-4000-8000-000000000812'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000813'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000814'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000815'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000816'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000817'::uuid, '00000000-0000-4000-8000-000000000812'::uuid),
    ('00000000-0000-4000-8000-000000000820'::uuid, '00000000-0000-4000-8000-000000000821'::uuid),
    ('00000000-0000-4000-8000-000000000821'::uuid, '00000000-0000-4000-8000-000000000823'::uuid),
    ('00000000-0000-4000-8000-000000000822'::uuid, '00000000-0000-4000-8000-000000000827'::uuid),
    ('00000000-0000-4000-8000-000000000823'::uuid, '00000000-0000-4000-8000-000000000824'::uuid),
    ('00000000-0000-4000-8000-000000000824'::uuid, '00000000-0000-4000-8000-000000000825'::uuid),
    ('00000000-0000-4000-8000-000000000825'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000826'::uuid, '00000000-0000-4000-8000-000000000822'::uuid),
    ('00000000-0000-4000-8000-000000000827'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000830'::uuid, '00000000-0000-4000-8000-000000000834'::uuid),
    ('00000000-0000-4000-8000-000000000831'::uuid, '00000000-0000-4000-8000-000000000837'::uuid),
    ('00000000-0000-4000-8000-000000000832'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000833'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000834'::uuid, '00000000-0000-4000-8000-000000000835'::uuid),
    ('00000000-0000-4000-8000-000000000835'::uuid, '00000000-0000-4000-8000-000000000836'::uuid),
    ('00000000-0000-4000-8000-000000000836'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000837'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000838'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000839'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000840'::uuid, '00000000-0000-4000-8000-000000000841'::uuid),
    ('00000000-0000-4000-8000-000000000841'::uuid, '00000000-0000-4000-8000-000000000843'::uuid),
    ('00000000-0000-4000-8000-000000000842'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000843'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000844'::uuid, NULL::uuid),
    ('00000000-0000-4000-8000-000000000845'::uuid, NULL::uuid)
) AS p (id, next)
WHERE e.id = p.id AND e.stable_id IS NULL;
