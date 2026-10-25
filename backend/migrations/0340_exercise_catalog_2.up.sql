-- Training planning (M9, docs/domains/training-planning.md): the second global exercise catalog.
-- It adds the rows the planner needs for a unit plan (Lösen, Arbeiten, Entspannen) in every
-- library: dressage, cavaletti, jumping, groundwork, lunge, hack and western. Global rows only
-- (stable_id NULL), IDs 00000000-0000-4000-8000-000000000NNN in the ranges of the spec:
-- dressage 850-889, cavaletti 890-905, jumping 906-919, groundwork 920-939, lunge 940-949,
-- hack 950-969, western 970-989; 990-999 stay reserved. The rules of the catalog (values, coverage,
-- how to add a row, sources) are in docs/domains/exercise-catalog.md and checked by
-- backend/internal/trainingapi/catalog_test.go.
--
-- Nothing is made up: every row names the pages it is based on in the sources column (and in the
-- comment above it), and every figure in a step (distances, counts, minutes, heights) stands on
-- one of those pages; a step without a figure in the sources leaves it out. The wording of the
-- steps follows the ISES learning principles used in the FN guidelines: one aid at a time, a light
-- signal first, the release named, praise instead of punishment.
--
-- Main sources: the FN member magazine "Pferd und Mensch" (series "10 Tipps" 2019-2025, "Lektion im
-- Fokus", training and jumping articles), the FN Merkblätter for the badges Bodenarbeit and Longieren
-- (§ 504 LPO heights come from the FN jumping article), the EWU Regelbuch 2024 and the Kreinberg
-- articles for western riding, and for the Kondition chain the FN articles plus the training-science
-- slides of Burger (Universität Bern, 2019).
-- Levels as in 0250: beginner = basic training and class E/A, intermediate = A* to L*, advanced = L** and
-- up; groundwork and lunge follow the FN badge stages (Reitabzeichen 10-8, 7-5 and Stufe 1, Stufe 2; LA 5,
-- LA 4, LA 3/2); western follows the EWU Westernreitlehre (Grundausbildung, Gymnastizierung, Manöver).
-- Equipment keys are the fixed list of the spec: facility equipment poles, cavaletti, jumps, obstacles;
-- personal equipment lunge, side_reins, double_lunge, kappzaum, tarp, umbrella, cones, flag, whip,
-- neck_ring, trailer. Rows with facility equipment have an alternative without it in the same library.
-- A row with one of these IDs is updated if it exists (global rows only); later corrections need a new
-- migration.

INSERT INTO exercises (id, stable_id, discipline, level, goal_tags, title, summary, steps, phase, minutes,
                       intensity, requires, equipment, suits, sources) VALUES

-- --- dressage ---
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
-- https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf
('00000000-0000-4000-8000-000000000850', NULL, 'dressage', 'beginner', '{losgelassenheit,takt,mobilitaet}', 'Schritt am hingegebenen Zügel', 'Ausgiebiges Schrittreiten bringt den Organismus in Schwung und schafft die Basis für Takt und Losgelassenheit.',
   '[
     "Vor jeder intensiven Arbeit im fleißigen Mittelschritt reiten, am besten mit hingegebenem Zügel: mindestens zehn Minuten, besser etwa 15 Minuten.",
     "Mit der Hand der Nickbewegung folgen, damit das Pferd den Ganaschenwinkel öffnen und die Stirn-Nasenlinie vor die Senkrechte nehmen kann.",
     "Das Wohlfühltempo des Pferdes finden: einige Tritte aussitzen, in gut gewähltem Tempo sitzt es sich meist gut.",
     "Währenddessen den eigenen Sitz lockern, zum Beispiel mit Lockerungs- und Dehnübungen im Schritt.",
     "Pferden mit Steifheit, etwa durch Arthrose, und im Winter mehr Zeit zum Warmlaufen geben."
   ]', 'warmup', 15, 'light', '{}', '{}', '{young,senior}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/,https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/
('00000000-0000-4000-8000-000000000851', NULL, 'dressage', 'beginner', '{losgelassenheit,takt,biegung}', 'Leichttraben auf großen Linien', 'Große Linien und Leichttraben lösen das Pferd, ohne den Rücken zu belasten.',
   '[
     "Nach dem Schritt mit Leichttraben beginnen: Das rhythmische Aufstehen und Einsitzen entlastet den Pferderücken und hilft, den Zweitakt zu erfühlen.",
     "Große Linien reiten: Zirkel, ganze Bahn, weite Schlangenlinien; als Faustregel die ersten 20 Minuten nur auf 20-Meter-Zirkeln.",
     "Bei jungen Pferden zu Beginn eher geradeaus und auf großen Bögen reiten.",
     "Das Tempo am Wohlfühltempo des Pferdes ausrichten, denn wird das Pferd zu Beginn über Tempo geritten, drohen Verspannungen und Taktstörungen.",
     "Regelmäßig die Hand wechseln und alle Übungen auf beiden Händen reiten."
   ]', 'warmup', 10, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/
('00000000-0000-4000-8000-000000000852', NULL, 'dressage', 'beginner', '{losgelassenheit,durchlaessigkeit,schwung}', 'Trab-Galopp-Übergänge zum Lösen', 'Wechsel zwischen Arbeitstrab und Arbeitsgalopp lockern die großen Muskelgruppen und fördern die Rückentätigkeit.',
   '[
     "Aus dem Arbeitstrab im Arbeitstempo angaloppieren, den Galopp nur in kurzen Reprisen reiten und ruhig zum Trab durchparieren.",
     "Vor jedem Übergang die neue Gangart beginnen wollen und energisch weitertreiben, so bleibt der Reiter im Fluss.",
     "Beim Durchparieren ausatmen und nachgeben; ein ruhiges, langgezogenes „Brrrh“ hat denselben Effekt.",
     "Auf beiden Händen reiten und zwischendurch kurz Zügel aus der Hand kauen lassen."
   ]', 'any', 10, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/
-- https://www.land-oberoesterreich.gv.at/Mediendateien/Formulare/Dokumente%20SGD%20Abt_ESV/Volkblattartikel_Wenn%20Pferde%20in%20die%20Jahre%20kommen_19.11.2022.pdf
('00000000-0000-4000-8000-000000000853', NULL, 'dressage', 'beginner', '{losgelassenheit,mobilitaet}', 'Schritt am langen Zügel zum Abschluss', 'Ruhig im Schritt ausgehen: Der Körper erholt sich aktiv und das Pferd verarbeitet das Gelernte.',
   '[
     "Nach der Arbeit mindestens zehn Minuten im Schritt reiten, am besten mit hingegebenem oder langem Zügel.",
     "Mit der Hand der Nickbewegung folgen, sodass das Pferd Kopf und Hals selbst wählen kann.",
     "Wenn möglich draußen ausgehen, notfalls zu Fuß mit dem Pferd an der Hand: Das macht die Pferde zufriedener.",
     "Nach einem intensiven Training folgt am besten eine Aktivpause, etwa leichtes Longieren, oder eine Passivpause wie Weidegang."
   ]', 'cooldown', 10, 'light', '{}', '{}', '{young,senior}', '{https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/,https://www.land-oberoesterreich.gv.at/Mediendateien/Formulare/Dokumente%20SGD%20Abt_ESV/Volkblattartikel_Wenn%20Pferde%20in%20die%20Jahre%20kommen_19.11.2022.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/
('00000000-0000-4000-8000-000000000854', NULL, 'dressage', 'beginner', '{biegung,takt,geraderichten}', 'Zirkel an Kegeln', 'Kegel an den Zirkelpunkten machen die Linie sichtbar und zeigen, ob der Zirkel wirklich rund ist.',
   '[
     "Vier Kegel an den vier Zirkelpunkten aufstellen.",
     "Im Arbeitstrab außen um die Kegel herum reiten und den Blick immer zum nächsten Punkt richten, nicht auf den Pferdehals.",
     "In jeder Ecke Takt, Tempo und Gleichgewicht halten und die Biegung um den inneren Schenkel fördern.",
     "Zum Steigern innen an den Kegeln entlang den Zirkel korrekt verkleinern.",
     "Auf beiden Händen reiten."
   ]', 'main', 10, 'light', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2022/
('00000000-0000-4000-8000-000000000855', NULL, 'dressage', 'beginner', '{schwung,rhythmus,durchlaessigkeit,versammlung}', 'Tempounterschiede im Trab', 'Zulegen und Aufnehmen in kurzen Reprisen aktiviert das Hinterbein und macht das Pferd durchlässig.',
   '[
     "Aus einem fleißigen Arbeitstrab leicht zulegen und wieder aufnehmen wie bei einer Ziehharmonika; zu Beginn reichen einige verlängerte Tritte.",
     "Das Tempo zurückführen, bevor das Pferd aus dem Gleichgewicht kommt und Taktfehler macht: lieber zweimal an einer langen Seite gut zulegen als eine ganze lange Seite auf der Vorhand.",
     "Beim Aufnehmen nicht langsamer reiten, sondern die Dynamik durch das Treiben erhalten.",
     "Beim Zulegen die Hand tief halten und den Rahmen größer werden lassen.",
     "Zulegen und Aufnehmen an wechselnden Stellen reiten: auf dem Zirkel, durch die kurze Seite, auf der Diagonalen oder dem zweiten Hufschlag."
   ]', 'main', 10, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2020/
('00000000-0000-4000-8000-000000000856', NULL, 'dressage', 'beginner', '{durchlaessigkeit,versammlung,aufmerksamkeit,balance}', 'Ganze Parade: Halten am Punkt', 'Die ganze Parade bringt das Hinterbein näher unter den Schwerpunkt und fördert die Lastaufnahme.',
   '[
     "Zuerst viele Übergänge und Tempounterschiede reiten, damit das Pferd die halben Paraden durchlässig annimmt.",
     "Die ersten ganzen Paraden aus dem Schritt und auf dem ersten Hufschlag reiten; klappt das, aus dem Trab oder auf der freien Linie ohne Hilfe der Bande.",
     "Mit dem Schenkel zur Hand treiben: Die Hand wirkt annehmend oder aushaltend, aber niemals rückwärts.",
     "Geht das Pferd auf der Vorhand, lieber weiterreiten und zuerst die Selbsthaltung verbessern.",
     "Das Pferd einen Moment länger stehen lassen, bis es ausatmet und entspannt; dann loben und weiterreiten."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/
('00000000-0000-4000-8000-000000000857', NULL, 'dressage', 'beginner', '{versammlung,durchlaessigkeit,schwung,aufmerksamkeit}', 'Energisch antraben nach wenigen Schritten', 'Energisch antraben, durchparieren, wenige Schritte gehen: Das schließt das Pferd und bereitet versammelnde Arbeit vor.',
   '[
     "Aus dem Schritt energisch antraben; der erste Trabtritt ist gleich schwungvoll: lostraben statt antraben.",
     "Nach wenigen Trabtritten durchparieren, wenige Schritte im Schritt gehen und wieder energisch antraben.",
     "Vor jedem Übergang die neue Gangart beginnen wollen, so bleibt der Reiter im Fluss und treibt weiter.",
     "Übergänge auch einmal am Punkt reiten, markiert mit Kegeln.",
     "Beim Durchparieren ausatmen und nachgeben."
   ]', 'main', 10, 'normal', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/
('00000000-0000-4000-8000-000000000858', NULL, 'dressage', 'beginner', '{biegung,versammlung,losgelassenheit}', 'Ecken als Viertelvolten', 'Jede Ecke ist eine Viertelvolte: Das Pferd wird aufgenommen, das Hinterbein aktiviert, die Biegung verbessert.',
   '[
     "Jede Ecke als Viertelvolte ausreiten und das Pferd darin aufnehmen.",
     "Die Bande begrenzt nach außen, die innere Hand kann leicht werden; die Biegung entsteht um den inneren Schenkel.",
     "Vor der Diagonalen noch einmal tief in die Ecke reiten, das schafft Platz zum Abwenden.",
     "Halbe Paraden helfen, in der Ecke Tempo und Takt zu sichern."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-ueberstreichen-05-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2021/
('00000000-0000-4000-8000-000000000859', NULL, 'dressage', 'beginner', '{anlehnung,balance,losgelassenheit}', 'Überstreichen', 'Überstreichen zeigt, ob das Pferd sich selbst trägt und nicht auf die Hand stützt.',
   '[
     "Mit halben Paraden vorbereiten und sicherstellen, dass das Pferd an den Hilfen steht und an das Gebiss herantritt.",
     "Für ein bis drei Pferdelängen beide Zügelfäuste entlang des Mähnenkamms etwa zwei Handbreit nach vorne schieben, sodass der Zügel durchhängt; der Oberkörper bleibt aufrecht.",
     "Das Pferd behält dabei Selbsthaltung, Takt und Tempo bei; die Nase kommt etwas deutlicher vor die Senkrechte.",
     "Danach die Zügel wieder aufnehmen und das Pferd loben, zum Beispiel mit einem leisen „Brav“ oder Klopfen am Hals.",
     "Hebt sich das Pferd heraus, zurück zum Anfang und das Annehmen der halben Paraden verbessern; auf gebogenen Linien mit der inneren Hand überstreichen."
   ]', 'main', 10, 'light', '{00000000-0000-4000-8000-000000000801}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-ueberstreichen-05-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/
('00000000-0000-4000-8000-000000000860', NULL, 'dressage', 'beginner', '{anlehnung,balance,aufmerksamkeit}', 'Reiten mit einer Hand', 'Beide Zügel in einer Hand zeigen schnell, wie fein die Verbindung zum Pferdemaul wirklich ist.',
   '[
     "Beide Zügel in eine Hand nehmen: Das führt das Gebiss ruhiger und leichter.",
     "Im Schritt und Arbeitstrab auf geraden und gebogenen Linien reiten und den Sitz unabhängig von der Hand halten.",
     "Zeigen sich Fehler oder Schwierigkeiten, an Takt, Losgelassenheit und am Treiben von hinten nach vorne arbeiten.",
     "Zwischendurch Zügel aus der Hand kauen lassen und überstreichen; das fördert eine feine Verbindung."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2025/
('00000000-0000-4000-8000-000000000861', NULL, 'dressage', 'beginner', '{anlehnung,losgelassenheit,balance}', 'Wechsel von Arbeits- und Dehnungshaltung', 'Der Wechsel zwischen Arbeits- und Dehnungshaltung verbessert Gleichgewicht, Losgelassenheit und Selbsthaltung.',
   '[
     "In der Arbeitshaltung das Genick als höchsten Punkt tragen lassen, in der Dehnungshaltung den Hals aus dem Widerrist heraus nach vorwärts-abwärts fallen lassen.",
     "Zügel-aus-der-Hand-kauen-Lassen in kurzen Reprisen reiten: Die Lektion wird begonnen und beendet und ist kein Dauerzustand.",
     "Zwischen hingegebenem Zügel, langem Zügel und Reiten am Zügel wechseln; so zeigt sich die Qualität der Anlehnung.",
     "Ein Mischmasch aus beiden Haltungen vermeiden.",
     "Beim Aufnehmen der Zügel das Pferd von hinten nach vorne an die weiche Hand herantreiben."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/
('00000000-0000-4000-8000-000000000862', NULL, 'dressage', 'beginner', '{takt,rhythmus,balance}', 'Takt halten in Wendungen und Übergängen', 'Im richtigen Tempo und mit halben Paraden bleibt der Takt auch in Wendungen und bei Übergängen erhalten.',
   '[
     "Das passende Tempo der Gangart finden: Zu hohes Tempo macht das Pferd eilig und verspannt, zu niedriges lässt es matt werden.",
     "In Wendungen und bei Übergängen halbe Paraden geben, um das Tempo durch die gute Abstimmung der Hilfen zu regulieren und den Takt zu sichern.",
     "Länge im Hals zulassen, denn der Hals dient dem Pferd als Balancierstange.",
     "Bleiben Taktstörungen bestehen, die Gesundheit des Pferdes abklären lassen."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
('00000000-0000-4000-8000-000000000863', NULL, 'dressage', 'beginner', '{takt,rhythmus,losgelassenheit,mobilitaet}', 'Schritt mit Fleiß und Raumgriff', 'Takt, Fleiß und Raumgriff im Schritt zeigen die Qualität der Ausbildung.',
   '[
     "Zuerst gut lösen: Nur das zufriedene und lockere Pferd kommt gelassen zum Schreiten.",
     "Den Schritt wechselseitig treiben: Wenn das rechte Hinterbein vorfußt, treibt der rechte Schenkel, links entsprechend.",
     "Mit der Hand die Nickbewegung zulassen, damit der Ganaschenwinkel offen und die Stirn-Nasenlinie vor der Senkrechten bleibt.",
     "Aktiv aus dem Schenkel unterstützen bei weicher Hand und ruhigem Oberkörper, ohne das Pferd anzuschieben.",
     "Planvoll und konzentriert reiten, vom längeren zum kürzeren Zügel, auf geraden und gebogenen Linien."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
('00000000-0000-4000-8000-000000000864', NULL, 'dressage', 'beginner', '{schwung,durchlaessigkeit,kraft,losgelassenheit}', 'Trab-Galopp-Wechsel für die Rückentätigkeit', 'Häufige Wechsel zwischen Dreitakt und Zweitakt verändern die Rückentätigkeit und kräftigen den Rücken.',
   '[
     "Häufig zwischen Arbeitstrab und Arbeitsgalopp wechseln; das Pferd verändert zwischen Dreitakt und Zweitakt die Rückentätigkeit.",
     "Die Übergänge flüssig von hinten nach vorne über den Rücken reiten.",
     "Pferde mit wenig Gehfreude werden durch das Wechselspiel frischer und motivierter.",
     "Zwischendurch im Schritt mit hingegebenem Zügel entspannen lassen, denn zur Entwicklung der Schubkraft gehört auch Entspannung."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000852}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/
-- https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/
('00000000-0000-4000-8000-000000000865', NULL, 'dressage', 'beginner', '{geraderichten,aufmerksamkeit,takt}', 'Zweiter Hufschlag und Viertellinie', 'Wer die eingefahrenen Wege verlässt, merkt, ob das Pferd wirklich in der Spur geht.',
   '[
     "Bewusst auf dem zweiten Hufschlag oder der Mittel- oder Viertellinie reiten statt immer an der Bande.",
     "Das Anlehnen an die Bande vermeiden: Es ist für das Pferd verlockend, weil es weniger anstrengt.",
     "Den Blick eine Pferdelänge vorausschicken, das verhilft zu geraden Linien und runden Zirkeln.",
     "Auf beiden Händen reiten; die schwierigere Seite öfter, aber mit regelmäßigen Pausen."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/,https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/
('00000000-0000-4000-8000-000000000866', NULL, 'dressage', 'beginner', '{biegung,geraderichten,balance}', 'Achten und halbe Volten', 'Volten, halbe Volten und Achten schulen Biegung und Geraderichten auf beiden Händen.',
   '[
     "Halbe Volten und doppelte halbe Volten exakt reiten; sie prüfen die Geschicklichkeit des Reiters und fördern das Gefühl für das Geraderichten.",
     "Volten, halbe Volten und Achten auf beiden Händen reiten, nicht nur auf der Schokoladenseite.",
     "Im Scheitelpunkt den Blick zum Ausgangspunkt wenden.",
     "Ein Kegel in der Mitte der Volte hilft bei der Orientierung.",
     "Die Größe nach Ausbildungsstand und Beweglichkeit wählen; in den meisten Dressurprüfungen werden Zehn-Meter-Volten verlangt."
   ]', 'main', 10, 'light', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2020/
('00000000-0000-4000-8000-000000000867', NULL, 'dressage', 'beginner', '{geraderichten,seitengaenge,biegung}', 'Schultervor', 'Schultervor bringt die Vorhand vor die Hinterhand und richtet das Pferd gerade; es bereitet das Schulterherein vor.',
   '[
     "Auf der hohlen Seite die Schulter so weit nach innen bringen, dass sich das Pferd hufschlagdeckend bewegt; ein Blick in den Spiegel zeigt den Erfolg.",
     "Das Einreiten im Schultervor hält das innere Hinterbein in der Spur, zum Beispiel vor dem Halten.",
     "Auf beiden Händen und auch unabhängig von der Bande reiten, bevor mit dem Schulterherein begonnen wird.",
     "Weniger ist mehr: nur wenige gute Tritte reiten, dann geradeaus; lässt der Trab nach, auf einen Zirkel abwenden."
   ]', 'main', 10, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
('00000000-0000-4000-8000-000000000868', NULL, 'dressage', 'beginner', '{seitengaenge,aufmerksamkeit,gelassenheit,durchlaessigkeit}', 'Schenkelweichen zur Konzentration', 'Schenkelweichen lenkt ein aufgeregtes Pferd auf die Reiterhilfen und fordert Konzentration und Koordination.',
   '[
     "Ist das Pferd aufgeregt oder abgelenkt, im Schritt oder Arbeitstrab schenkelweichen; das fördert die Akzeptanz der Reiterhilfen.",
     "Mit leichter Stellung, aber ohne Biegung reiten: Der innere Schenkel treibt knapp hinter dem Gurt vorwärts-seitwärts, der äußere verwahrt.",
     "Zu Beginn den der Bande zugewandten Schenkel weichen lassen; die Bande begrenzt, und der Reiter kann sich auf das Vorwärts-Seitwärts konzentrieren.",
     "Die Abstellung unter 45 Grad halten und das Vorwärts in den Vordergrund stellen.",
     "Fließend zwischen Schenkelweichen und Geradeausreiten wechseln, sobald das Pferd aufmerksam an den Hilfen steht."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
('00000000-0000-4000-8000-000000000869', NULL, 'dressage', 'beginner', '{seitengaenge,durchlaessigkeit,takt,geraderichten}', 'Schenkelweichen im Arbeitstrab', 'Im Arbeitstrab verlangt das Schenkelweichen mehr Takt und Fleiß und verfeinert die Reaktion auf den Schenkel.',
   '[
     "Erst im Mittelschritt sicher schenkelweichen, dann auch im Arbeitstrab; das Vorwärts steht immer im Vordergrund.",
     "Die Abstellung unter 45 Grad halten, um Takt und Fleiß zu erhalten; zu viel Abstellung führt oft zum Ausweichen über die Schulter.",
     "Fortgeschrittene Paare reiten auch ohne Bande: die Vorhand ins Bahninnere führen, die Hinterhand bleibt auf dem ersten Hufschlag, der innere Schenkel lässt weichen.",
     "Schenkelweichen und Übertreten-Lassen fördern Schenkelgehorsam und Durchlässigkeit.",
     "Später an jeder beliebigen Stelle schenkelweichen, auch im Gelände."
   ]', 'main', 10, 'normal', '{00000000-0000-4000-8000-000000000804}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2022/
-- https://magazin.pferdesport-deutschland.de/project/so-klappt-es-mit-dem-abteilungsreiten-06-2019/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
('00000000-0000-4000-8000-000000000870', NULL, 'dressage', 'beginner', '{gelassenheit,vertrauen,aufmerksamkeit,durchlaessigkeit}', 'Abteilungsreiten', 'Dressurreiten in der Gruppe prüft Gehorsam und Gelassenheit und schult Abstände, Tempo und exakte Hufschlagfiguren.',
   '[
     "Mit festgelegter Reihenfolge reiten; der Têtenreiter geht voran, die anderen halten ein bis zwei Pferdelängen Abstand.",
     "Auf Kommandos achten: Auf das Ankündigungskommando folgt eine kurze Pause zur Vorbereitung, dann das Ausführungskommando.",
     "Neigt das Pferd im Galopp zur Spannung, zuerst im Trab üben, bis es gelassen bleibt; galoppiert wird zunächst einzeln, dann zu zweit und erst dann in der ganzen Abteilung.",
     "Die kurzen Seiten nutzen, um Abstände auszugleichen: tiefer in die Ecke reiten, um Abstand zu gewinnen, oder die Ecke abrunden, um aufzuschließen.",
     "Als Sicherheitshilfe über die Ohren des Pferdes hinweg gerade noch die Sprunggelenke des Vorderpferdes sehen."
   ]', 'main', 15, 'normal', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2022/,https://magazin.pferdesport-deutschland.de/project/so-klappt-es-mit-dem-abteilungsreiten-06-2019/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
('00000000-0000-4000-8000-000000000871', NULL, 'dressage', 'beginner', '{gelassenheit,vertrauen,aufmerksamkeit,balance}', 'Gelassen halten und stehen', 'Gelassenes Halten will geübt werden: Das Pferd atmet aus, entspannt und wartet auf die nächste Hilfe.',
   '[
     "Das Pferd vor dem Halten im Gleichgewicht und geschlossen halten lassen.",
     "Bewusst einen Moment länger stehen lassen, bis das Pferd deutlich ausatmet und entspannt; das ist der Moment zum Loben.",
     "Danach mit einem klaren Signal weiterreiten; das Pferd wartet auf die Hilfe zum Anreiten.",
     "Auch beim Aufsitzen, Absitzen, Nachgurten und Bügelverstellen das Pferd zuerst zu gelassenem Halten im Gleichgewicht bringen.",
     "Das Pferd soll ruhig stehen bleiben, wenn die Zügel in eine Hand genommen werden."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
('00000000-0000-4000-8000-000000000872', NULL, 'dressage', 'beginner', '{gelassenheit,vertrauen,aufmerksamkeit}', 'Trainingsort wechseln', 'Wechselnde Orte und neue optische Reize halten Pferde auch auf dem Turnier und in fremder Umgebung gelassen.',
   '[
     "Den Trainingsort regelmäßig wechseln: Halle, Reitplatz, Gelände oder ein fremder Platz.",
     "In die Halle Pylonen, Ständer oder Tonnen stellen und die Arbeit darum herum gestalten; das gewöhnt an visuelle Reize und fördert die Konzentration.",
     "Ein triebiges Pferd wird durch die Reize oft motivierter, ein nervöses fokussierter.",
     "Dem Pferd Zeit geben, ungewohnte Objekte zunächst skeptisch zu betrachten.",
     "Auch einmal aufladen und auf fremden Plätzen üben, das gibt Pferd und Reiter Sicherheit auf dem Turnier."
   ]', 'main', 20, 'light', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/,https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
('00000000-0000-4000-8000-000000000873', NULL, 'dressage', 'beginner', '{kondition,ausdauer,takt,rhythmus}', 'Trabreprisen im Arbeitstempo', 'Längere Strecken im Arbeitstrab bauen die Grundkondition auf: erst die Dauer steigern, dann die Intensität.',
   '[
     "Je nach Ausgangslage zunächst nur längere Strecken im Arbeitstempo traben.",
     "Von anfänglich zehn Minuten schrittweise steigern und zuerst die Dauer verlängern, bevor die Intensität steigt.",
     "Zwei- bis dreimal pro Woche Konditionstraining in den Trainingsplan einbauen.",
     "Zeichen von Übermüdung wie Stolpern oder Bewegungsverzögerungen nicht ignorieren und die Belastung anpassen."
   ]', 'main', 10, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/
('00000000-0000-4000-8000-000000000874', NULL, 'dressage', 'beginner', '{kondition,ausdauer,kraft}', 'Galoppreprisen im frischen Arbeitsgalopp', 'Galoppreprisen steigern die Ausdauer und kräftigen die Bauchmuskeln, wenn zuerst der Umfang wächst und nicht das Tempo.',
   '[
     "Erst nach dem Trabtraining erste Galoppreprisen einbauen.",
     "In einem frischen Arbeitsgalopp beginnen und vor allem den Umfang steigern, nicht das Tempo.",
     "Als Ziel zum Beispiel zweimal fünf Minuten mit einer Pause dazwischen.",
     "Der Galopp lässt das Pferd Sprung für Sprung die Bauchmuskulatur anspannen.",
     "Im leichten Sitz mit kurzem Bügel entlastet der Reiter den Rücken des Pferdes."
   ]', 'main', 15, 'demanding', '{00000000-0000-4000-8000-000000000873}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/
('00000000-0000-4000-8000-000000000875', NULL, 'dressage', 'beginner', '{kondition,ausdauer,kraft,balance}', 'Frischer Galopp im leichten Sitz', 'Längeres Galoppieren in frischem Tempo trainiert den ganzen Organismus und besonders die Rumpfmuskulatur.',
   '[
     "Im leichten Sitz mit kurzem Bügel galoppieren; der Reiter entlastet den Pferderücken und hält ihm den Rücken frei.",
     "In frischem Tempo reiten, das Pferd galoppiert dabei gern vorwärts; das Tempo darf auch einmal flotter ausfallen.",
     "Die Galoppreprisen kurz halten und dazwischen Trab- oder Schrittpausen einbauen.",
     "Immer in die Bewegungsrichtung sitzen: vermehrt auf dem inneren Gesäßknochen, innere Schulter zurück, Blick zum nächsten Zirkelpunkt."
   ]', 'main', 10, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/
('00000000-0000-4000-8000-000000000876', NULL, 'dressage', 'beginner', '{turnier,durchlaessigkeit,aufmerksamkeit}', 'Einreiten, Halten und Grüßen', 'Schnurgerades Einreiten, geschlossenes Halten und sicheres Grüßen sind die Visitenkarte der Aufgabe.',
   '[
     "Die Grußaufstellung in Teilen üben: abwenden und schnurgerade im geforderten Tempo geradeaus reiten.",
     "Durchlässig zum Halten übergehen und ruhig geschlossen halten; das Pferd wartet auf die Hilfe zum Anreiten.",
     "Aus dem ausbalancierten Halten schnurgerade und mit direktem Antritt anreiten.",
     "In Übungsfolgen aufbauen: Übergänge an der Bande, dann auf der freien Linie, dann Abwenden und Mittellinie; Kegel helfen bei der Orientierung.",
     "Das Pferd soll ruhig stehen bleiben, auch wenn die Zügel in eine Hand genommen werden und gegrüßt wird."
   ]', 'main', 10, 'light', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
('00000000-0000-4000-8000-000000000877', NULL, 'dressage', 'beginner', '{turnier,aufmerksamkeit,durchlaessigkeit}', 'Aufgabe in Baustücken üben', 'Einzelne Teile der Aufgabe üben, bis sie sitzen, statt immer die ganze Aufgabe von vorn zu beginnen.',
   '[
     "Aus der Aufgabe der Klasse Baustücke herausnehmen und einzeln üben, auch auf dem zweiten oder dritten Hufschlag, der Viertel- oder der Mittellinie.",
     "Erst wenn die einzelnen Lektionen sicher sitzen, die ganze Aufgabe reiten; die schnelle Abfolge der Lektionen wird oft unterschätzt, schon ab Klasse E.",
     "Klappt etwas nicht, nicht immer die gesamte Aufgabe neu beginnen und nicht die Lektionen wiederholen, die ohnehin funktionieren.",
     "Lektionen sind Zeichen des Ausbildungsstandes: Probleme darin führen zu einem Schritt zurück und besseren Grundlagen; etwa zehn Prozent des Trainings gelten den Lektionen der Aufgabe.",
     "Die Aufgabe abgehen oder im Schritt reiten, damit Zeit bleibt zu überlegen, welche Hilfe wann nötig ist."
   ]', 'main', 15, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
('00000000-0000-4000-8000-000000000878', NULL, 'dressage', 'beginner', '{turnier,durchlaessigkeit,aufmerksamkeit}', 'Aufgabe der Klasse E durchreiten', 'Die Aufgabe der Klasse E aus dem FN-Aufgabenheft einmal ganz durchreiten und danach auswerten.',
   '[
     "Die Aufgabe der Klasse E aus dem aktuellen FN-Aufgabenheft auswendig lernen: zu Hause abgehen, aufmalen oder sich vorsprechen lassen, nicht vorlesen.",
     "Das Pferd zuerst auf den Punkt lösen, dann die Aufgabe an einem Stück durchreiten, ohne zu stoppen, wenn etwas nicht klappt.",
     "Jede Ecke wie eine Viertelvolte reiten und jede Hufschlagfigur korrekt von Punkt zu Punkt; Schenkelweichen kommt in vielen Aufgaben der Klassen E und A vor.",
     "Klappt eine Lektion nicht, ruhig weiterreiten und erst nach dem Ritt auswerten.",
     "Richter achten auf Sitz und Einwirkung, geregelte Grundgangarten, das Einhalten der Hufschlagfiguren und Harmonie."
   ]', 'main', 20, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
('00000000-0000-4000-8000-000000000879', NULL, 'dressage', 'beginner', '{turnier,durchlaessigkeit,aufmerksamkeit}', 'Aufgabe der Klasse A durchreiten', 'Die Aufgabe der Klasse A aus dem FN-Aufgabenheft an einem Stück durchreiten und Schwächen erkennen.',
   '[
     "Die Aufgabe der Klasse A aus dem aktuellen FN-Aufgabenheft auswendig lernen und die Wege vorher zu Fuß abgehen.",
     "Übergänge am Punkt reiten, denn zum Beispiel das Angaloppieren exakt an der Bahnmarke bestätigt präzise Einwirkung und gute Durchlässigkeit.",
     "Zehn-Meter-Volten und Schlangenlinien korrekt reiten; Bahnfiguren zeigen, dass das Pferd sicher an den Hilfen steht.",
     "Die Aufgabe ohne Stopp durchreiten; kleine Fehler hinter sich lassen und den Blick auf die kommenden Lektionen richten.",
     "Danach auswerten, welche Baustücke noch Übung brauchen, und diese einzeln üben."
   ]', 'main', 20, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/
('00000000-0000-4000-8000-000000000880', NULL, 'dressage', 'beginner', '{turnier,losgelassenheit,aufmerksamkeit,durchlaessigkeit}', 'Vorbereitungsplatz: Abreiten wie zu Hause', 'Auf dem Vorbereitungsplatz wird das Pferd gezielt gelöst und eingestimmt; geübt wird zu Hause.',
   '[
     "Zu Hause auf die Uhr schauen: wissen, wie lange Pferd und Reiter zum Warmwerden brauchen, und zehn Minuten Puffer für Unvorhergesehenes einplanen.",
     "Die Arbeit gestalten wie zu Hause: Aufsitzen, Schritt reiten, lösende Arbeit; geht das Pferd nach der Lösungsphase nochmal Schritt am langen Zügel, bekommt es das auch auf dem Turnier.",
     "Nur Basisarbeit reiten: Trab-Galopp-Übergänge, Zügel-aus-der-Hand-kauen-Lassen und Überstreichen; im letzten Drittel vor dem Start einzelne Lektionen, ohne sich mit dem Pferd anzulegen.",
     "Lektionen, die dem Pferd schwerfallen, nicht wieder und wieder abfragen; das bringt nur Spannung und Stress.",
     "Läuft etwas nicht wie geplant, ruhig bleiben und auf Übergänge, Wendungen, Schenkelweichen oder Rückwärtsrichten setzen."
   ]', 'main', 20, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/}'),
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
('00000000-0000-4000-8000-000000000881', NULL, 'dressage', 'beginner', '{turnier,aufmerksamkeit,durchlaessigkeit}', 'Generalprobe unter Prüfungsbedingungen', 'Etwa 14 Tage vor dem Turnier die Aufgabe wie im Turnier reiten: mit Klingel, Richter am Tisch und fester Uhrzeit.',
   '[
     "Etwa 14 Tage vor dem Turniertag eine Generalprobe der Aufgabe machen, gern mit Klingel, mit Richter am Tisch, mit Platzwechsel und mit festgelegter Uhrzeit.",
     "Die Abläufe des Turniertages im Training genau üben: Aufsitzen, Abreiten, Einreiten und Grüßen.",
     "Die Aufgabe reiten, ohne bei Fehlern zu stoppen, und danach gemeinsam auswerten, was gelingt und woran noch gearbeitet wird.",
     "In der verbleibenden Zeit bis zur Prüfung das Training abwechslungsreich gestalten und weiter an den Schwächen arbeiten."
   ]', 'main', 30, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/
('00000000-0000-4000-8000-000000000882', NULL, 'dressage', 'beginner', '{turnier,gelassenheit,vertrauen,aufmerksamkeit}', 'Viereck zum eigenen Viereck machen', 'Das Pferd lernt das Prüfungsviereck kennen: außen herumreiten, Linien abgehen, am Richterhäuschen ruhig vorbeikommen.',
   '[
     "Auf dem Turniergelände zuerst eine Geländebegehung machen: Wo darf geritten werden, wie sieht das Prüfungsviereck aus, wie funktioniert die Logistik.",
     "Die Linien abgehen und Dinge von außen, zum Beispiel Werbetafeln oder Blumenkübel, als Orientierung nutzen.",
     "Vor dem Einreiten einmal außen herum reiten, am besten auf beiden Händen, die Ecken ausreiten und mit guter Vorwärtstendenz.",
     "Will das Pferd an einer Stelle nicht vorbei, es nicht hineinschauen lassen: schenkelweichartig vorbeireiten, Kopf weg vom Richterhaus, ruhig bleiben und weiterreiten.",
     "Am Ende der Aufgabe kann noch einmal am Häuschen vorbeigeritten werden, um positiv zu enden."
   ]', 'main', 15, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/}'),
-- https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
('00000000-0000-4000-8000-000000000883', NULL, 'dressage', 'intermediate', '{turnier,durchlaessigkeit,versammlung,aufmerksamkeit}', 'Aufgabe der Klasse L durchreiten', 'Die Aufgabe der Klasse L aus dem FN-Aufgabenheft durchreiten und die neuen Lektionen sicher einbauen.',
   '[
     "Die Aufgabe der Klasse L aus dem aktuellen FN-Aufgabenheft auswendig lernen und die Lektionen vorher einzeln als Baustücke üben.",
     "Der einfache Galoppwechsel und der Außengalopp werden ab Klasse L verlangt; beide setzen ein gewisses Maß an Versammlung voraus.",
     "Zehn-Meter-Volten sicher reiten, später auch Volten mit acht Metern Durchmesser.",
     "Die Aufgabe an einem Stück durchreiten; gelingt eine Lektion einmal nicht, gibt es dafür eine schwächere Note, doch der Rest zeigt trotzdem den Ausbildungsstand, deshalb ruhig weiterreiten.",
     "Zu Hause sicher auf dem Niveau der Aufgabe sein; bei einem hartnäckigen Problem lieber eine Klasse tiefer starten."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000815,00000000-0000-4000-8000-000000000816}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2020/
('00000000-0000-4000-8000-000000000884', NULL, 'dressage', 'intermediate', '{schwung,rhythmus,durchlaessigkeit}', 'Tempounterschiede im Galopp', 'Zulegen und Aufnehmen im Galopp aktiviert die Hinterbeine und fördert lebhaftes Abfußen.',
   '[
     "Schon in der Lösungsphase viele Übergänge zwischen Arbeitstrab und Arbeitsgalopp einbauen und den Galopp nur in kurzen Reprisen fordern.",
     "Gefühlvoll herausgeritten zulegen und aufnehmen, auf beiden Händen sowie auf geraden und gebogenen Linien.",
     "Besonders in der Rückführung nicht langsamer reiten, sondern die Dynamik durch das Treiben erhalten.",
     "Für mehr Raumgriff den Rahmen erweitern lassen: Länge im Hals durch nachgiebige Zügelführung und treibende Unterstützung.",
     "Bergauf eher zulegen, bergab eher aufnehmen wollen, wenn Hügel zur Verfügung stehen."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000855}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-05-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2024/
('00000000-0000-4000-8000-000000000885', NULL, 'dressage', 'advanced', '{durchlaessigkeit,balance,versammlung}', 'Fliegender Galoppwechsel', 'Der fliegende Wechsel prüft die Durchlässigkeit und verlangt einen ausbalancierten, geradegerichteten Galopp.',
   '[
     "Voraussetzung ist ein ausbalancierter, geradegerichteter, bergauf springender Galopp: Der Wechsel ist nur so gut wie der Galopp davor.",
     "Die Hilfe zum Umspringen im Moment der Einbeinstütze vor der Schwebephase geben: beide Schenkel gleichzeitig umlegen und die neue innere Hüfte leicht nach vorne bringen.",
     "Die neue innere Hand gibt im Moment des Wechsels weich nach und lässt den Galoppsprung nach vorne heraus.",
     "Die ersten Wechsel zur besseren Galoppseite üben; erst wenn das Pferd dort sicher auf die Hilfe umspringt, auch zur schwierigeren Seite.",
     "Einen guten Wechsel sofort loben; einen fehlerhaften Wechsel niemals bestrafen, sonst wird die Lektion zur Angstlektion."
   ]', 'main', 15, 'demanding', '{00000000-0000-4000-8000-000000000815}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-05-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-11-12-2023/
('00000000-0000-4000-8000-000000000886', NULL, 'dressage', 'advanced', '{versammlung,kraft,durchlaessigkeit}', 'Galopp-Pirouette', 'Die Galopp-Pirouette fordert und fördert hohe Lastaufnahme und Hankenbeugung im versammelten Galopp.',
   '[
     "Nur im versammelten Galopp reiten; ganze Pirouetten mit sechs bis acht Galoppsprüngen, halbe mit drei bis vier.",
     "Zu Trainingszwecken Arbeitspirouetten mit größerem Kreisbogen und ohne vorgegebene Zahl an Galoppsprüngen reiten.",
     "Die Einleitung erfolgt im Schultervor: Daraus die Vorhand um das innere Hinterbein führen und die Hinterhand auf kleinem Kreisbogen zum fleißigen Unterspringen anregen.",
     "Die innere Hand fühlt in jedem Galoppsprung weich vor und ermöglicht das Unterspringen.",
     "Geht der klare Galoppsprung verloren, sofort aus der Pirouette herausreiten und mit Tempounterschieden im Galopp oder Zirkel-Verkleinern den Galopp wiederherstellen."
   ]', 'main', 15, 'demanding', '{00000000-0000-4000-8000-000000000807,00000000-0000-4000-8000-000000000815}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-11-12-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2022/
('00000000-0000-4000-8000-000000000887', NULL, 'dressage', 'advanced', '{schwung,versammlung,kraft,durchlaessigkeit}', 'Kadenz durch Zulegen und Aufnehmen', 'Dynamisches Versammeln mit Zulegen und Aufnehmen in kurzen Reprisen ist die Basis für Kadenz.',
   '[
     "Halbe Paraden von hinten nach vorne im Zusammenspiel aller Hilfen reiten; sie schließen die Hinterhand heran und lassen mehr Last aufnehmen, Fleiß und Gehfreude bleiben erhalten.",
     "Durch gefühlvolles Zulegen und Aufnehmen in kurzen Reprisen kräftigen; bei der Rückführung energisch weitertraben.",
     "Der Trab bleibt schwungvoll und nicht langsam: kürzere, aber energisch abfußende Tritte.",
     "Die Anforderungen schrittweise steigern, ins Pferd hineinhorchen und rechtzeitig Pausen einbauen; Versammlung ist körperlich anstrengend, vergleichbar mit der Kniebeuge beim Menschen."
   ]', 'main', 10, 'demanding', '{00000000-0000-4000-8000-000000000814,00000000-0000-4000-8000-000000000855}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2022/}'),

-- --- cavaletti ---
-- https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/
('00000000-0000-4000-8000-000000000890', NULL, 'cavaletti', 'beginner', '{takt,vertrauen,aufmerksamkeit,balance}', 'Cavaletti im Schritt kennenlernen', 'Ein einzelnes Cavaletti im Schritt: Das Pferd lernt ruhig und entspannt geradeaus darüber zu schreiten.',
   '[
     "Kennt das Pferd noch keine Cavaletti, zunächst im Schritt mit einem einzelnen Cavaletti an der Bande vertraut machen; Fangständer können ein Ausweichen verhindern.",
     "Das Cavaletti gerade und mittig anreiten und das Pferd mit den Hilfen einrahmen; die Hand lässt dem Pferd genügend Raum, sich auszubalancieren.",
     "So lange üben, bis das Pferd ruhig und entspannt geradeaus über das Cavaletti schreitet.",
     "Ein erfahrenes Führpferd kann besonders großen Skeptikern helfen, ihre Scheu zu überwinden.",
     "Nach und nach die Anzahl der Cavaletti steigern."
   ]', 'main', 10, 'light', '{}', '{cavaletti}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/,https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/
-- https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
('00000000-0000-4000-8000-000000000891', NULL, 'cavaletti', 'intermediate', '{aufmerksamkeit,balance,rhythmus,takt}', 'Trab-Cavaletti mit Zwischentritt', 'Ein Zwischentritt in der Cavaletti-Reihe fördert Aufmerksamkeit, Balance und das Gefühl für taktmäßige Bewegungen.',
   '[
     "Vier Cavaletti mit einem Abstand von etwa 1,30 Metern an der langen Seite aufbauen.",
     "Ein Cavaletti aus der Reihe herausnehmen, sodass ein Zwischentritt entsteht, oder Cavaletti in unterschiedlicher Höhe aufstellen; das fördert Aufmerksamkeit und Balance.",
     "Zur Überprüfung gleichmäßig schwungvoll und rhythmisch weitertraben.",
     "Bei einem Pferd mit vielen kleinen Zwischentritten zuerst die Losgelassenheit und Dehnungsbereitschaft verbessern: Beginnt man zu früh, entzieht es sich der Trittverlängerung."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000821}', '{cavaletti}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/,https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/
('00000000-0000-4000-8000-000000000892', NULL, 'cavaletti', 'beginner', '{losgelassenheit,anlehnung,mobilitaet,takt}', 'Dehnung über Trab-Cavaletti', 'Über niedrigen Cavaletti dehnt sich das Pferd vermehrt vorwärts-abwärts und verbessert seine Losgelassenheit.',
   '[
     "Das Pferd sollte vor den Cavaletti schon gelöst sein und sichere Dehnungsbereitschaft beim Zügel-Verlängern zeigen.",
     "Niedrige Cavaletti von etwa 15 Zentimetern Höhe wählen und im Trab mittig anreiten.",
     "Über den Cavaletti die Zügel aus der Hand kauen lassen; die Hand geht in Richtung Pferdemaul leicht mit, soweit die Dehnung des Halses es erfordert, ohne die Verbindung aufzugeben.",
     "Grundtempo und Rhythmus bleiben vor, über und nach dem Cavaletti erhalten; der Blick geht nach vorne."
   ]', 'main', 10, 'light', '{00000000-0000-4000-8000-000000000821}', '{cavaletti}', '{}', '{https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/,https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
('00000000-0000-4000-8000-000000000893', NULL, 'cavaletti', 'intermediate', '{aufmerksamkeit,biegung,balance,mobilitaet}', 'Das kreative Kreuz', 'Vier Stangen im Kreuz wechseln Geradeausreiten und gebogene Linien und fördern Aufmerksamkeit und Beweglichkeit.',
   '[
     "In der Mitte der Bahn vier Bodenricks, niedrige Cavaletti oder Stangen auf Blöcken zu einem Kreuz zusammenlegen: zwei auf der Mittellinie, zwei auf der Verbindungslinie zwischen E und B.",
     "Im Trab über eine der vier Stangen geradeaus reiten, nach außen auf eine Volte abwenden und aus der Volte heraus über die nächste Stange reiten, bis ein vierblättriges Kleeblatt entsteht.",
     "In einer 20 Meter breiten Halle etwas weiter innen über die Stangen reiten, damit die Volten nicht zu klein werden.",
     "Zum Steigern ohne Volte direkt auf einem kleinen Zirkel über alle vier Stangen traben oder galoppieren: zuerst weiter außen, dann nach und nach verkleinern; für den Galopp die Cavaletti erhöhen.",
     "Nach einigen Wiederholungen durch energisches Geradeausreiten auf der ganzen Bahn die Anforderung wechseln."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000823}', '{cavaletti}', '{}', '{https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/
('00000000-0000-4000-8000-000000000894', NULL, 'cavaletti', 'beginner', '{aufmerksamkeit,geraderichten,durchlaessigkeit,balance}', 'Stangengasse für präzise Linien und Übergänge', 'Zwei Stangen hintereinander geben dem Auge Fixpunkte und erziehen zu exakten Linien und Übergängen am Punkt.',
   '[
     "Auf der Mittellinie, parallel zur kurzen Seite, jeweils zwei Stangen auf Blöcken oder niedrige Cavaletti mit etwa 1,30 Metern Abstand hintereinanderlegen: auf Höhe der Zirkelpunkte und exakt über X.",
     "Mit der einfachsten Version beginnen: bei A oder C einen Zirkel anlegen und exakt durch die Stangengasse bei X reiten; danach einen Mittelzirkel mit den Gassen auf Höhe der Zirkelpunkte.",
     "Im Arbeitstrab und Arbeitsgalopp in der Lösungsphase reiten.",
     "Die Gasse für präzise Übergänge nutzen: zwischen den Stangen, je nach Ausbildungsstand, aus dem Arbeitstrab zum Mittelschritt oder zum Halten durchparieren.",
     "Das Pferd sicher einrahmen, damit die Linien exakt gelingen; regelmäßiger Handwechsel unterstützt das Geraderichten."
   ]', 'any', 15, 'light', '{}', '{poles}', '{}', '{https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/
-- https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/
('00000000-0000-4000-8000-000000000895', NULL, 'cavaletti', 'intermediate', '{rhythmus,durchlaessigkeit,balance,aufmerksamkeit}', 'Galoppsprünge variieren auf 15 Metern', 'Zwei Cavaletti mit drei, vier oder fünf Galoppsprüngen reiten schult Rhythmus und Gleichmaß.',
   '[
     "Zwei Cavaletti mit 15 Metern Abstand an der langen Seite aufstellen.",
     "Wahlweise mit drei, vier oder fünf Galoppsprüngen dazwischen reiten; das gelingt nur, wenn sich der Galopp sicher verlängern und verkürzen lässt.",
     "Ein Gefühl für den eigenen Rhythmus des Pferdes entwickeln: Ist die Distanz zu kurz, ist das Tempo zu hoch, ist sie zu lang, ist das Tempo zu niedrig.",
     "Die Galoppsprünge laut mitzählen, damit der Rhythmus überprüft werden kann."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000824}', '{cavaletti}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/,https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
('00000000-0000-4000-8000-000000000896', NULL, 'cavaletti', 'intermediate', '{takt,balance,losgelassenheit,mobilitaet}', 'Longieren über Cavaletti', 'Über Cavaletti lernt das Pferd ohne Reitergewicht, seine Bewegungen zu koordinieren und den Rücken aufzuwölben.',
   '[
     "Cavaletti mit abgeflachten oder abgerundeten Seitenteilen verwenden, damit die Longe nicht hängen bleibt.",
     "In der Reithalle an der kurzen Seite aufbauen: zunächst davor und daran vorbei longieren und erst darüber, wenn das Pferd bereit ist.",
     "Die Abstände an das Pferd anpassen: im Schritt etwa 0,80 Meter, im Trab etwa 1,30 Meter, im Galopp etwa drei Meter.",
     "Mit Cavaletti nicht versuchen, ein heißes Pferd zu bremsen: Die Verletzungsgefahr ist groß.",
     "Im Galopp die Cavaletti an den Zirkelpunkten aufstellen und das Pferd über jedes einzeln springen lassen."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000821}', '{cavaletti,lunge}', '{}', '{https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/,https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/
-- https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/
('00000000-0000-4000-8000-000000000897', NULL, 'cavaletti', 'intermediate', '{kondition,kraft,rhythmus,balance}', 'Cavaletti-Reihe für Kraft und Kondition', 'Eine Reihe aus bis zu fünf Cavaletti kräftigt Rücken und Hinterhand und steigert Kondition und Ausdauer.',
   '[
     "Eine Reihe mit bis zu vier oder fünf Cavaletti aufbauen; bei mehr als fünf fällt es den meisten Pferden schwer, genügend Kraft und Koordination aufzubringen.",
     "Zuerst im Schritt, dann im Trab und schließlich im Galopp über die Reihe arbeiten: Die niedrigste Einstellung ist für alle Gangarten geeignet, die mittlere für Trab und Galopp, die höchste für den Galopp.",
     "Das Pferd hebt die Beine aktiver an und fußt mit der Hinterhand mehr unter den Schwerpunkt; das kräftigt den Rücken und fördert die Beweglichkeit.",
     "In kurzen Reprisen arbeiten und dazwischen im Schritt am hingegebenen Zügel entspannen lassen."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000821}', '{cavaletti}', '{}', '{https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/,https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/
('00000000-0000-4000-8000-000000000898', NULL, 'cavaletti', 'beginner', '{takt,rhythmus,losgelassenheit,balance}', 'Rhythmus ohne Stangen: Galoppsprünge zählen', 'Wo keine Stangen liegen, schult das Mitzählen den Rhythmus: erst im Galopp ohne Sprung, dann mit halben Paraden im Takt.',
   '[
     "Zunächst ohne Sprung in der Galopparbeit immer wieder einige Galoppsprünge mitzählen, und zwar in dem Augenblick, in dem das Pferd nacheinander beide Vorderbeine aufsetzt.",
     "Das passende Tempo finden: dynamisch und fleißig, aber sicher an den Hilfen und vor dem Reiter; zu hohes Tempo macht das Pferd eilig und verspannt.",
     "Rhythmisches Galoppieren erleichtert die Atmung und das konzentrierte Arbeiten.",
     "In Wendungen und bei Übergängen halbe Paraden geben, um Tempo und Takt zu sichern."
   ]', 'any', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/
('00000000-0000-4000-8000-000000000899', NULL, 'cavaletti', 'beginner', '{aufmerksamkeit,geraderichten,durchlaessigkeit,balance}', 'Präzise Linien mit Kegeln statt Stangen', 'Kegel zeigen dem Auge den Weg: Linien, Zirkelpunkte und Übergänge am Punkt gelingen auch ohne Stangen.',
   '[
     "Übergänge bewusst am Punkt reiten statt sie dem Pferd zu überlassen; Kegel markieren den Punkt.",
     "Kegel an den Zirkelpunkten zeigen schnell, wer sicher auf gebogenen Linien reitet.",
     "Zwei Kegel links und rechts der Mittellinie als Törchen erleichtern das Einteilen von Volten und geben einen Orientierungspunkt für die ganze Parade.",
     "Die Linien mit dem Blick zum nächsten Punkt reiten, nicht auf den Pferdehals schauen."
   ]', 'any', 10, 'light', '{}', '{cones}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/}'),

-- --- jumping ---
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2021/
('00000000-0000-4000-8000-000000000906', NULL, 'jumping', 'beginner', '{springgymnastik,balance,aufmerksamkeit}', 'Leichter Sitz ohne Sprung', 'Ein ausbalancierter, losgelassener leichter Sitz ist die Voraussetzung fürs Springen und schult Gleichgewicht und Bewegungsgefühl.',
   '[
     "Die Bügel für den leichten Sitz etwa 3 bis 6 Loch kürzer verschnallen als für das Dressurreiten; lieber mit wenigen Löchern anfangen und langsam weiter verkürzen.",
     "Zum Einnehmen des leichten Sitzes die Hüfte nach hinten schieben, die Füße unter den Knien halten und den Oberkörper tief lassen, wie bei einer Kniebeuge; mit federndem Fußgelenk nah am Pferd elastisch mitschwingen.",
     "Alles, was im Dressursitz klappt, auch mit kurzem Bügel reiten: Übergänge, Wendungen und Lektionen auf ebenem Hufschlag, auf geraden und gebogenen Linien und in allen Grundgangarten.",
     "In die Bewegungsrichtung schauen und Wendungen mit dem Körper und dem Blick einleiten; der Zügel dient nicht primär zum Lenken.",
     "Zu Beginn kann ein Halsriemen Halt geben, damit das Pferdemaul verschont bleibt: Ein zusätzlicher Bügelriemen oder ein Führstrick reicht dafür aus."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/
('00000000-0000-4000-8000-000000000907', NULL, 'jumping', 'beginner', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit,balance}', 'Imaginäres Parcoursreiten', 'Ein Parcours ohne Sprünge übt Linien, Übergänge und Rhythmus und prägt Parcourslinien ein, ganz ohne Aufbau.',
   '[
     "Eine Parcourslinie ohne Sprünge festlegen und Punkt für Punkt einprägen: Beim imaginären Parcoursreiten treffen verschiedene Übungen der dressurmäßigen Vorbereitung zusammen.",
     "Vorgegebene Linien und korrekte Wege einhalten und dabei Übergänge, Handwechsel und Wendungen kombinieren.",
     "Fließende Übergänge innerhalb des Galopps üben: Nur wer die Länge der Galoppsprünge jederzeit verändern kann, bewältigt die Aufgaben eines Parcours rhythmisch.",
     "Den Blick schon zum nächsten Punkt der Linie richten, denn der Blick leitet die Bewegungen des Körpers und gibt dem Pferd die Richtung vor.",
     "Zum Abschluss Schrittpausen mit hingegebenem Zügel einbauen und das Pferd kurz entspannen lassen."
   ]', 'main', 15, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/ein-sprung-viele-moeglichkeiten-11-12-2021/
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
-- https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017/
('00000000-0000-4000-8000-000000000908', NULL, 'jumping', 'beginner', '{springgymnastik,balance,aufmerksamkeit}', 'Vom Kreuz zum Steilsprung und Oxer', 'Aus einem einfachen Kreuz wächst Schritt für Schritt ein Steilsprung und ein Hochweitsprung: mittig, ruhig und gerade.',
   '[
     "Mit einem Kreuzsprung beginnen, denn es erzieht Pferd und Reiter, in der Mitte zu bleiben; etwa 2,50 Meter davor ein niedriges Cavaletti oder Bodenrick legen und aus ruhigem, kontrolliertem Tempo von einer langen Linie im Leichttraben anreiten.",
     "Nach der Landung das Pferd sofort loben und auf geradem Weg vom Galopp über den Trab in den Schritt übergehen; bei jungen oder unerfahrenen Pferden Fangständer verwenden, denn sie geben dem Pferd eine optische Orientierungshilfe und verhindern ein seitliches Ausbrechen.",
     "Aus dem Kreuz einen niedrigen Steilsprung mit Vorlegestange aufbauen: Das Cavaletti etwa 10 bis 20 Zentimeter weiter vom Sprung weglegen.",
     "Für das Springen aus dem Galopp das vorgelegte Cavaletti etwa drei Meter vor das Hindernis legen.",
     "Im nächsten Schritt den Steilsprung zum Hochweitsprung umbauen und die Vorlegestange geringfügig dichter heranlegen; Weg, Tempo, Rhythmus und der gleichmäßig weiche Kontakt zum Pferdemaul bleiben gleich."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000826}', '{jumps,poles}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/ein-sprung-viele-moeglichkeiten-11-12-2021/,https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/,https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
('00000000-0000-4000-8000-000000000909', NULL, 'jumping', 'beginner', '{turnier,aufmerksamkeit,durchlaessigkeit}', 'Parcours besichtigen und Linien planen', 'Wer den Parcours sorgfältig abgeht, plant Linien, Distanzen und Galoppsprünge schon vor dem Start.',
   '[
     "Erst nach Freigabe des Parcours durch die Richtergruppe besichtigen und etwa fünf bis zehn Minuten einplanen; die Skizze zeigt unter anderem die erlaubte Zeit sowie Start und Ziel.",
     "Von Start bis Ziel die Linie abgehen: Wie reite ich das Hindernis an, wo liegt die ideale Linie zwischen den Sprüngen und wie viele Galoppsprünge sind dazwischen zu reiten?",
     "Die Distanzen mit gleichmäßiger Schrittlänge abgehen und in den Wendungen die Galoppsprünge abmessen, um einzuschätzen, ob ein Galoppsprung mehr oder weniger zu reiten ist.",
     "Die Startlinie genauso ernst nehmen wie den ersten Sprung und besondere Herausforderungen für das eigene Pferd einplanen, etwa ein erster Sprung vom Ausgang weg.",
     "Bei einer zweifachen Kombination, erkennbar an den Buchstaben A und B, auch auf die Hindernistypen achten, denn ob ein Oxer oder ein Steilsprung das erste Element ist, wirkt sich auf das Anreiten aus."
   ]', 'main', 10, 'light', '{}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/,https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/
-- https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
('00000000-0000-4000-8000-000000000910', NULL, 'jumping', 'beginner', '{turnier,aufmerksamkeit,gelassenheit}', 'Einreiten, Grüßen und Start', 'Einreiten und Grüßen sind die Visitenkarte des Reiters: ruhig halten, deutlich grüßen, erst nach dem Zeichen starten.',
   '[
     "Lage von Einritt, Richterturm und Anzeigentafel sowie den Weg zum Vorbereitungsplatz vorher anschauen; auch die Startlinie gehört zur Planung.",
     "Einreiten und Grüßen üben: sicher halten, den Blick zu den Richtern richten, deutlich grüßen und erst nach dem Glockenzeichen losreiten.",
     "Das Pferd soll ruhig und lange genug stehen, auch wenn die Zügel in eine Hand genommen werden und gegrüßt wird; ruhiges Stehen sofort loben.",
     "Die Startlinie genauso wichtig nehmen wie den ersten Sprung: Wo ist der Start, und wie komme ich am besten dahin?",
     "Wendungen und den Weg vor und nach dem ersten Sprung sorgfältig planen, denn die Linienführung ist entscheidend für gutes Anreiten."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/,https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
('00000000-0000-4000-8000-000000000911', NULL, 'jumping', 'beginner', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Hindernisfolge wie im Parcours', 'Zwei Hindernisse mit drei bis sechs Galoppsprüngen dazwischen üben die Distanz so, wie sie im Parcours gefordert wird.',
   '[
     "Mit einer Folge von zwei Hindernissen beginnen, die im passenden Abstand auf gerader oder gebogener Linie stehen; zwischen den beiden Hindernissen braucht das Pferd drei bis sechs Galoppsprünge.",
     "Die Folge im gleichmäßigen Tempo und mit der gewünschten Anzahl von Galoppsprüngen reiten und die Galoppsprünge gelegentlich mitzählen.",
     "Schon über dem ersten Hindernis das nächste im Blick haben, um den Weg präzise einzuhalten.",
     "Nach der Landung dem Pferd beim ersten Galoppsprung erlauben, sich an die nachgebende Hand heranzudehnen, und es dann von hinten an die Hand heranreiten; der letzte Galoppsprung gehört dem Pferd, die Einwirkung endet beim vorletzten.",
     "Klappt die Folge, bei weiten Distanzen die Galoppsprünge verlängern, bei engen verkürzen oder einen Galoppsprung mehr oder weniger reiten; bei fünf oder mehr Galoppsprüngen besteht eher die Chance auf einen Ausgleich."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000908}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
('00000000-0000-4000-8000-000000000912', NULL, 'jumping', 'beginner', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Parcours in E-Höhe', 'Ein ganzer Parcours in E-Höhe zeigt, ob Linien, Rhythmus und Gelassenheit vor dem ersten Start sitzen.',
   '[
     "Einen Trainingsparcours in E-Höhe aufbauen: 80 bis 85 Zentimeter, in der Halle mindestens 6 Sprünge, auf dem Außenplatz mindestens 8, höchstens eine zweifache Kombination; jedes Hindernis fair und einladend mit klarer Grundlinie und klarer oberer Begrenzungslinie.",
     "Den Parcours vorher abgehen und die Linie einprägen; bei den ersten Hindernisfolgen lange, gerade Wege mit genug Zeit zur Vorbereitung auf den nächsten Sprung wählen.",
     "Gleichmäßig und rhythmisch galoppieren: Das Pferd bleibt dynamisch und fleißig, sicher an den Hilfen und vor dem Reiter; die Galoppsprünge laut mitzählen.",
     "Verweigert das Pferd, nicht bestrafen, sondern Weg, Tempo, Rhythmus und Gleichgewicht prüfen: Die Vorbereitung war nicht gut; einen oder zwei Schritte zurückgehen und erst Sicherheit und Vertrauen schaffen.",
     "Nach dem letzten Sprung im Schritt mit hingegebenem Zügel ausreiten und rechtzeitig mit einem positiven Gefühl aufhören."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000911}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/,https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
('00000000-0000-4000-8000-000000000913', NULL, 'jumping', 'beginner', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Parcours in A-Höhe', 'Ein Parcours in A-Höhe verlangt eine gleichmäßige Verbindung zum Pferdemaul und einen geschmeidig angepassten leichten Sitz.',
   '[
     "Einen Trainingsparcours in A-Höhe aufbauen: 90 bis 105 Zentimeter, in der Halle mindestens 6 Sprünge, auf dem Außenplatz mindestens 9, im A** höchstens zwei zweifache Kombinationen.",
     "Den Parcours vorher abgehen und die Linien zwischen den Sprüngen planen; auf Hindernisse ohne klare Konturen achten, denn einfarbige Sprünge sind für Pferde deutlich schwieriger einzuschätzen.",
     "Eine gleichmäßige Verbindung zum Pferdemaul halten und den leichten Sitz geschmeidig anpassen: ruhig sitzen, ruhig treiben, nicht schieben.",
     "Zur Orientierung zählt das vorgeschriebene Tempo: in der Halle 300 bis 350 Meter pro Minute, im Freien 350 bis 400 Meter pro Minute; die Ausschreibung kann es anders regeln.",
     "Nach dem Parcours im Schritt am hingegebenen Zügel ausreiten, damit das Pferd entspannen kann."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000912}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/,https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/
('00000000-0000-4000-8000-000000000914', NULL, 'jumping', 'intermediate', '{springgymnastik,durchlaessigkeit,aufmerksamkeit,rhythmus}', 'Zweifache Kombination', 'Zwei Hindernisse mit einem oder zwei Galoppsprüngen dazwischen fordern Konzentration und Reaktionsvermögen von Pferd und Reiter.',
   '[
     "Eine Kombination besteht aus zwei oder mehr Hindernissen, deren Abstand so gewählt ist, dass das Pferd den Weg zwischen Landung und Absprung mit einem oder zwei Galoppsprüngen schafft.",
     "Mit einem In-and-Out aus zwei Kreuzen beginnen; der Abstand beträgt etwa 3,30 bis 3,40 Meter, die Angaben sind Richtwerte und werden an die Größe des Pferdes und die Höhe der Hindernisse angepasst.",
     "Das erste Hindernis richtig anreiten: Weg, Grundtempo, Rhythmus, Schwung und Absprungdistanz so beeinflussen, dass dem Pferd auch für den Aussprung eine gute Bewegung möglich ist; das Pferd bleibt im Gleichgewicht vor dem Reiter und an den treibenden Hilfen.",
     "Steht die Kombination eher weit, helfen ein höheres Grundtempo und eine normal passende Absprungdistanz; eine enge Kombination verlangt ein verkürztes, deutlich versammeltes Tempo und eine dichtere Absprungdistanz.",
     "Sind zweifache Kombinationen sicher, auf dem Weg zur Klasse L auch dreifache einbeziehen, zum Einstieg aus einfach zu springenden Elementen."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000911}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/,https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/ein-sprung-viele-moeglichkeiten-11-12-2021/
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
('00000000-0000-4000-8000-000000000915', NULL, 'jumping', 'intermediate', '{springgymnastik,balance,aufmerksamkeit,durchlaessigkeit}', 'Springen aus der Wendung', 'Am Einzelsprung üben Pferd und Reiter, aus verschieden großen Wendungen ruhig und mittig anzureiten.',
   '[
     "Am Einzelsprung Situationen üben, die im Parcours wichtig sind: Springen aus unterschiedlich großen Wendungen, aus unterschiedlichen Tempi und Anreitewegen und das schräge Überwinden des Hindernisses.",
     "Beim Springen aus der Wendung stets die Mitte des Hindernisses anvisieren und das Pferd mit den äußeren Hilfen, also dem verwahrenden Schenkel und Zügel, sicher auf der Linie halten.",
     "Vom Leichten zum Schweren: Die Wendung zum Sprung nach und nach enger gestalten, sehr enge Wendungen aber vermeiden.",
     "Den guten Rhythmus des Galopps erhalten; das Pferd bleibt im Gleichgewicht und vor dem Reiter.",
     "Alle Übungen gleichmäßig von beiden Händen aus reiten und die Einheit rechtzeitig positiv beenden; ein erfahrener Ausbilder gibt Hinweise zu Sitz und Einwirkung."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000908}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/ein-sprung-viele-moeglichkeiten-11-12-2021/,https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/
-- https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2022/
-- https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017/
-- https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf
('00000000-0000-4000-8000-000000000916', NULL, 'jumping', 'beginner', '{losgelassenheit,takt,mobilitaet}', 'Lösen vor dem Springen', 'Eine sorgfältige Lösungsphase mit Schritt, Leichttraben und Übergängen schützt vor Verletzungen und bereitet auf die Springarbeit vor.',
   '[
     "Mindestens 15 Minuten im fleißigen Mittelschritt reiten, am besten mit hingegebenem Zügel; erst nach einer sorgfältigen Lösungsphase wird gesprungen; Pferden mit Steifheit, etwa durch Arthrose, mehr Zeit zum Warmlaufen geben.",
     "Mit Leichttraben auf großen Linien weiterarbeiten: Zirkel, ganze Bahn, weite Schlangenlinien; auf gute Bodenverhältnisse achten und sehr enge Wendungen vermeiden; junge Pferde brauchen ausreichend Zeit, sich mit den Materialien in der Halle vertraut zu machen.",
     "Häufig zwischen Arbeitstrab und Arbeitsgalopp wechseln, das fördert die Rückentätigkeit; Trab-Schritt-Übergänge bringen eiligen Pferden Ruhe.",
     "Zügel-aus-der-Hand-kauen-Lassen und Überstreichen fördern die Losgelassenheit; viele hohe Sprünge machen nur müde.",
     "Dehnt sich das Pferd vertrauensvoll an die Reiterhand heran und folgt bei länger werdendem Zügel nach vorwärts-abwärts, ist es gelöst; wenn nicht, weiter im Schritt lösen."
   ]', 'warmup', 15, 'light', '{}', '{}', '{young,senior}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/,https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2022/,https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017/,https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/
('00000000-0000-4000-8000-000000000917', NULL, 'jumping', 'beginner', '{losgelassenheit,mobilitaet}', 'Schritt nach dem Springen', 'Nach der Springarbeit entspannt der Schritt am langen Zügel und das Pferd verarbeitet das Gelernte.',
   '[
     "Nach der Springarbeit im Schritt mit hingegebenem Zügel ausreiten und das Pferd entspannen lassen; rechtzeitig einen positiven Abschluss finden.",
     "Das Pferd soll sich auch in der Pause weiter bewegen, zum Beispiel bei einem Spaziergang oder auf der Weide.",
     "Nach einem intensiven Training folgt entweder eine Passivpause wie Weidegang oder eine Aktivpause wie leichtes Longieren."
   ]', 'cooldown', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/,https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/
-- https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2022/
('00000000-0000-4000-8000-000000000918', NULL, 'jumping', 'intermediate', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Parcours in L-Höhe', 'Ein ganzer Parcours in L-Höhe fordert Rhythmus- und Distanzgefühl sowie Rittigkeit und Durchlässigkeit in Wendungen, Folgen und Kombinationen.',
   '[
     "Einen Trainingsparcours in L-Höhe aufbauen: 110 bis 115 Zentimeter, in der Halle mindestens 7 Sprünge, auf dem Außenplatz mindestens 9, höchstens zwei zweifache und eine dreifache Kombination.",
     "Den Parcours vorher abgehen; ab Klasse L verlangt der Parcours mehr Rhythmus- und Distanzgefühl vom Reiter.",
     "Ist die Wendung vor dem Sprung enger, wird das Anreiten anspruchsvoller: Das Pferd kommt zurück, lässt sich außen führen und galoppiert gleichmäßig an die Anlehnung heran.",
     "Nach einem guten Durchgang braucht es keine weitere Wiederholung; mit gutem Gefühl im Schritt am hingegebenen Zügel beenden und anstrengende Parcours dosiert einsetzen."
   ]', 'main', 25, 'demanding', '{00000000-0000-4000-8000-000000000913,00000000-0000-4000-8000-000000000914}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/,https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/,https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2022/
('00000000-0000-4000-8000-000000000919', NULL, 'jumping', 'advanced', '{turnier,rhythmus,aufmerksamkeit,durchlaessigkeit}', 'Parcours in M-Höhe', 'Ein ganzer Parcours in M-Höhe verzeiht keine unpassende Vorgabe und verlangt ein sicheres Gefühl für Absprungdistanz und Distanzen.',
   '[
     "Einen Trainingsparcours in M-Höhe aufbauen: 120 bis 135 Zentimeter, in der Halle mindestens 8 bis 9 Sprünge, auf dem Außenplatz mindestens 10 bis 11.",
     "Den Parcours vorher abgehen und die Distanzen mit gleichmäßiger Schrittlänge abmessen; ab Klasse M ist die passende Absprungdistanz ausschlaggebend, weil das Pferd sich aus ungünstigen Situationen nicht mehr heraushelfen kann.",
     "In engeren Distanzen kommt das Pferd nach der Landung am Einsprung zurück; das gibt es erst ab M* oder S, darunter baut man keine Zwischendistanzen.",
     "Anstrengende Parcours dosiert einsetzen und nach dem Parcours im Schritt am hingegebenen Zügel ausreiten."
   ]', 'main', 25, 'demanding', '{00000000-0000-4000-8000-000000000918}', '{jumps}', '{}', '{https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/,https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2022/}'),

-- --- groundwork ---
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/
('00000000-0000-4000-8000-000000000920', NULL, 'groundwork', 'beginner', '{takt,aufmerksamkeit,mobilitaet}', 'Schritt führen und Tempo wechseln', 'Ruhiges Führen im Schritt mit Tempowechseln löst das Pferd und stimmt es auf die Zusammenarbeit ein.',
   '[
     "Das Pferd von links neben der Schulter führen: Den Strick etwa 40 bis 50 Zentimeter unter dem Karabinerhaken fassen, das Strickende geordnet in der Hand halten und es nie um Hand oder Finger wickeln.",
     "Die Führposition zwischen Genick und Schulter beibehalten und gleichzeitig losgehen; das Pferd orientiert sich am Tempo des Menschen.",
     "Im Schritt das Gangmaß verkürzen und verlängern: Das Pferd folgt der Tempoveränderung des Führenden, der Strick hängt leicht durch.",
     "Das Anhalten über die Körpersprache einleiten; reagiert das Pferd nur bedingt, die Einwirkung angemessen verstärken, zum Beispiel mit einem Stimmsignal, dem Heben der linken Hand in Augenhöhe leicht vor dem Kopf und/oder leichten Impulsen mit dem Führstrick.",
     "Gewendet wird vom Menschen weg: Wer von links führt, wendet nach rechts und macht dabei größere Schritte als das Pferd."
   ]', 'warmup', 5, 'light', '{}', '{}', '{senior}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/}'),
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
-- https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/
-- https://www.land-oberoesterreich.gv.at/Mediendateien/Formulare/Dokumente%20SGD%20Abt_ESV/Volkblattartikel_Wenn%20Pferde%20in%20die%20Jahre%20kommen_19.11.2022.pdf
('00000000-0000-4000-8000-000000000921', NULL, 'groundwork', 'beginner', '{losgelassenheit,mobilitaet,vertrauen}', 'Spaziergang an der Hand', 'Ein ruhiger Spaziergang bringt Bewegung, Abwechslung und Vertrauen, auch als Pause nach intensiver Arbeit.',
   '[
     "Im Schritt am locker durchhängenden Strick spazieren gehen, möglichst draußen und auf unterschiedlichen Böden, denn verschiedene Böden fördern die Trittsicherheit.",
     "Natürliche Reaktionen wie Ohrenspiel oder ein kurzes Stocken zulassen und ruhig weitergehen.",
     "Anderen Pferden und Menschen mit möglichst großem Sicherheitsabstand begegnen und sich nach Möglichkeit von vorne begegnen.",
     "Als Aktivpause nach einem intensiven Training eignet sich ein Spaziergang, denn das Pferd soll sich auch in der Pause weiter bewegen."
   ]', 'any', 10, 'light', '{}', '{}', '{senior}', '{https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/,https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/,https://www.land-oberoesterreich.gv.at/Mediendateien/Formulare/Dokumente%20SGD%20Abt_ESV/Volkblattartikel_Wenn%20Pferde%20in%20die%20Jahre%20kommen_19.11.2022.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2021/
('00000000-0000-4000-8000-000000000922', NULL, 'groundwork', 'beginner', '{losgelassenheit,mobilitaet,vertrauen}', 'Ruhig ausgehen und abhalftern', 'Zum Abschluss ruhig im Schritt führen und das Pferd sicher entlassen: Das gibt dem Pferd Ruhe und dem Menschen Sicherheit.',
   '[
     "Zum Abschluss mindestens zehn Minuten im Schritt am durchhängenden Strick führen, wenn möglich draußen.",
     "Das Pferd für ruhiges Gehen und gutes Mitarbeiten loben; Loben bestärkt das Pferd, wenn es etwas gut gemacht hat.",
     "Beim Entlassen auf die Weide, in die Box oder in den Paddock das Pferd zuerst mit dem Kopf in Richtung Tür oder Tor drehen, bevor das Halfter abgenommen wird.",
     "So kann der Mensch bei ungestümem Loslaufen oder einem Freudenhüpfer des Pferdes nicht getroffen werden."
   ]', 'cooldown', 10, 'light', '{}', '{}', '{senior}', '{https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2021/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/
('00000000-0000-4000-8000-000000000923', NULL, 'groundwork', 'beginner', '{vertrauen,gelassenheit,aufmerksamkeit}', 'Ansprechen, Annähern und Stallgasse', 'Freundlich ansprechen, ruhig annähern und sicher durch die Stallgasse führen: Die Basis für Vertrauen im Alltag.',
   '[
     "Das Pferd zuerst ansprechen, sich möglichst von schräg vorne nähern, neben Kopf und Hals stehen bleiben und über Berührung Kontakt aufnehmen.",
     "Beim Aufhalftern darauf achten, dass das Pferd mithilft; mit festen Schuhen und Handschuhen führen.",
     "Auf der Stallgasse offene Türen, herumstehende oder herumliegende Gegenstände und die Bodenbeschaffenheit beachten.",
     "Bei der Begegnung mit anderen Pferden einen möglichst großen Sicherheitsabstand einhalten; in einer engen Stallgasse die Pferde nicht direkt aneinander vorbeiführen, sondern sich von vorne begegnen.",
     "Ein angebundenes Pferd nur mit einer Hilfsperson passieren."
   ]', 'main', 10, 'light', '{}', '{}', '{young}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
-- https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/
('00000000-0000-4000-8000-000000000924', NULL, 'groundwork', 'beginner', '{vertrauen,aufmerksamkeit}', 'Führen von beiden Seiten', 'Von links und von rechts führen macht Pferd und Mensch geschickter und sorgt in vielen Situationen für mehr Sicherheit.',
   '[
     "Klappen die Übungen auf der linken Seite gut, ist es Zeit für das Führen von der rechten; das verbessert die Geschicklichkeit von Mensch und Pferd.",
     "Auch auf der ungewohnten Seite gefühlvolle Signale geben und zwischen Pferdekopf und Schulter zügig mitgehen.",
     "Den Seitenwechsel zunächst im Halten üben.",
     "Später die Führseite im Gehen wechseln: vor dem Pferd mit Blickrichtung zum Pferd, ohne es in seiner Bewegung zu behindern; der Strick berührt dabei nicht den Boden."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/,https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
('00000000-0000-4000-8000-000000000925', NULL, 'groundwork', 'beginner', '{aufmerksamkeit,mobilitaet,takt}', 'Slalom und Gangmaßwechsel', 'Slalom und Tempowechsel schulen Konzentration und Geschmeidigkeit und machen das Pferd aufmerksam für feine Signale.',
   '[
     "Pylonen oder Kegel in einer Reihe aufstellen und zu Beginn größere Abstände wählen, damit der Ablauf harmonisch bleibt.",
     "Im Schritt durch den Slalom führen, ohne im Slalom die Führseite zu wechseln; Führposition, Stimme, die Drehung des Schultergürtels und ein richtungsweisender Arm zeigen den Weg.",
     "Zwischendurch das Gangmaß im Schritt verkürzen und verlängern: Das Pferd folgt der Tempoveränderung des Führenden.",
     "Für Geübte die Abstände später enger oder unterschiedlich groß wählen; die häufigen Richtungswechsel fördern Konzentration und Geschmeidigkeit."
   ]', 'main', 10, 'light', '{}', '{cones}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
-- https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/
('00000000-0000-4000-8000-000000000926', NULL, 'groundwork', 'beginner', '{seitengaenge,durchlaessigkeit,aufmerksamkeit,mobilitaet}', 'Hinterhand weichen lassen', 'Die Hinterhand weicht auf Körpersprache: Das macht das Pferd durchlässig und aufmerksam für feine Signale.',
   '[
     "Das Pferd ruhig stehen lassen, zum Beispiel angebunden, und sich neben die Schulter stellen.",
     "Mit auffordernder Körpersprache und Stimmhilfe zum Weichen auffordern; zusätzlich seitlich in Höhe des Oberschenkels oder am Rumpf hinter der Gurtlage berühren.",
     "Wenn nötig den Kopf des Pferdes festhalten und es entgegen der Bewegungsrichtung stellen, sodass es um die Vorhand herumtritt.",
     "Schon beim ersten Übertreten die Aufforderung beenden und loben: Das Nachlassen ist die Belohnung.",
     "Auf beiden Seiten üben."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/,https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/}'),
-- https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000927', NULL, 'groundwork', 'beginner', '{seitengaenge,durchlaessigkeit,aufmerksamkeit,mobilitaet}', 'Schulter weichen und folgen lassen', 'Wer die Schulter des Pferdes mit Körpersprache bewegen kann, gymnastiziert das Pferd und gewinnt seine Aufmerksamkeit.',
   '[
     "Sich mit Präsenz frontal auf die Pferdeschulter zubewegen und wenn nötig die Hände heben, bis die Schulter weicht; zunächst einen kleinen Umweg wählen und das Pferd mit der Vorhand folgen lassen.",
     "Zum Folgen das Pferd mit etwas Abstand am Strick führen, aus der Bewegung die Körpervorderseite zum Pferd drehen und die Gerte vor der Pferdebrust vorbei in Richtung der abgewandten Schulter führen.",
     "Sich dann im 90-Grad-Winkel rückwärts vom Pferd wegbewegen: Das Pferd dreht die Vorhand zu dir und kommt auf dich zu.",
     "Die Übung hilft auch, wenn das Pferd abgelenkt ist und sich wieder auf den Menschen konzentrieren soll; beim ersten Nachgeben sofort loben."
   ]', 'main', 10, 'light', '{}', '{whip}', '{}', '{https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/,https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
('00000000-0000-4000-8000-000000000928', NULL, 'groundwork', 'beginner', '{gelassenheit,vertrauen}', 'Regenschirm kennenlernen', 'Neugier statt Schreck: Am Regenschirm lernt das Pferd in kleinen Schritten, Unbekanntes gelassen zu untersuchen.',
   '[
     "Zuerst in der Halle oder auf einem umzäunten Platz üben; der Mensch geht neben dem Pferd auf der Seite, die zwischen dem Schreckreiz und dem Pferd liegt, damit der Fluchtweg nach vorwärts-seitwärts frei bleibt.",
     "Den geschlossenen Schirm zeigen und das Pferd daran schnuppern lassen; nimmt es ihn gelassen hin, mit dem Schirm am Hals beginnend langsam Richtung Kruppe berühren und dabei hinter dem Pferd deutlich außer Reichweite der Hufe bleiben.",
     "Den Schirm vor den Augen des Pferdes schließen und wieder öffnen; die Stärke des Reizes stufenweise immer nur so weit steigern, dass keine Fluchtreaktion ausgelöst wird.",
     "Jeden Schritt des Pferdes auf den Schirm zu mit einer Pause belohnen und die Neugier des Pferdes nutzen, damit es sich freiwillig nähert und das Objekt untersucht.",
     "Regt sich das Pferd auf, den Reiz verkleinern und ruhig bleiben; bei einem Erschrecken nicht bestrafen."
   ]', 'main', 15, 'light', '{}', '{umbrella}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/,https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
('00000000-0000-4000-8000-000000000929', NULL, 'groundwork', 'intermediate', '{gelassenheit,vertrauen,aufmerksamkeit}', 'Flattervorhang und Rappelsack', 'Flatterband und Rappelsack gewöhnen das Pferd an Geräusche und Bewegung: Gelassenheit in kleinen Schritten.',
   '[
     "Den Flattervorhang aus Flatterband zunächst in der Halle oder auf einem umzäunten Platz aufhängen und das Pferd erst nur schauen und neugierig untersuchen lassen.",
     "Den Rappelsack erst ruhig am Boden zeigen und dann stufenweise stärker bewegen, immer nur so weit, dass keine Fluchtreaktion ausgelöst wird.",
     "Neben dem Pferd auf der Seite gehen, die zwischen dem Schreckreiz und dem Pferd liegt.",
     "Natürliche Reaktionen wie Ohrenspiel, auf die Aufgabe gerichtete Aufmerksamkeit oder ein kurzes Stocken zulassen, wenn das Pferd die Aufgabe danach gelassen absolviert.",
     "Ziel ist, dass das Pferd mit leicht durchhängendem Strick zwanglos, fleißig und aufmerksam neben dem Menschen durch den Vorhang geht; danach loben."
   ]', 'main', 15, 'light', '{00000000-0000-4000-8000-000000000928}', '{obstacles}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/}'),
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000930', NULL, 'groundwork', 'intermediate', '{gelassenheit,vertrauen,balance}', 'Über die Brücke', 'Die Brücke schult Gelassenheit und Trittsicherheit: Das Pferd lernt, dem Menschen auf ungewohntem Untergrund zu folgen.',
   '[
     "Eine Brücke verwenden, die mindestens 90 Zentimeter breit und 250 Zentimeter lang ist; das sind die Mindestmaße der EWU-Trailaufgabe.",
     "Das Pferd zunächst geführt mit der Aufgabe vertraut machen, denn Trail-Aufgaben sind für das Pferd anspruchsvoll.",
     "Das Pferd die Brücke in Ruhe betrachten und untersuchen lassen und die Neugier nutzen, damit es sich freiwillig nähert.",
     "Im Schritt mit hingegebenem Strick hinüberführen, damit das Pferd seinen Hals als Balancierstange nutzen kann; jeden Fortschritt loben."
   ]', 'main', 15, 'light', '{00000000-0000-4000-8000-000000000928}', '{obstacles}', '{}', '{https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/,https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/,https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
('00000000-0000-4000-8000-000000000931', NULL, 'groundwork', 'intermediate', '{gelassenheit,vertrauen,aufmerksamkeit}', 'Engpass passieren', 'Durch einen Engpass führen übt das willige und gelassene Passieren und den sicheren Wechsel der Führposition.',
   '[
     "Einen Engpass zum Beispiel aus Strohballen oder anderen Hindernissen aufbauen, zunächst weit wie einen Trichter, und ihn mit wachsender Sicherheit des Pferdes verengen.",
     "Zum Passieren die Führposition vor das Pferd wechseln und das Pferd willig und gelassen hindurchführen.",
     "Jeden Schritt des Pferdes zum Engpass mit einer Pause belohnen und das Pferd bei Unsicherheit nicht mit Rucken am Strick bedrängen.",
     "Auf beiden Seiten üben."
   ]', 'main', 15, 'light', '{00000000-0000-4000-8000-000000000928}', '{obstacles}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
('00000000-0000-4000-8000-000000000932', NULL, 'groundwork', 'intermediate', '{aufmerksamkeit,balance,mobilitaet,vertrauen}', 'Stangenlabyrinth', 'Im Stangenlabyrinth schulen Pferd und Mensch Geschicklichkeit, Beweglichkeit, Konzentration und Koordination.',
   '[
     "Aus Stangen ein Labyrinth legen, zum Beispiel Mikado, Dreieck oder doppeltes L; die Stangen gegen Wegrollen sichern oder Schaumstoffstangen verwenden.",
     "Das Pferd im Schritt den vorgegebenen Weg innerhalb der Stangen führen, ohne dass es das Labyrinth seitlich verlässt oder Stangen berührt; die Position des Führenden darf dabei wechseln, auch außerhalb der Stangen.",
     "Ruhig arbeiten, jederzeit anhalten können und mit klarer Linienführung sowie eindeutigen und feinen Hilfen führen.",
     "Bei Schwierigkeiten den Schwierigkeitsgrad verringern und die Übung mit einer gelungenen Ausführung beenden."
   ]', 'main', 15, 'light', '{}', '{poles}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
-- https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/
('00000000-0000-4000-8000-000000000933', NULL, 'groundwork', 'intermediate', '{aufmerksamkeit,balance,mobilitaet}', 'Halten über der Stange', 'Exaktes Halten über einer Stange erfordert höchste Aufmerksamkeit und gute Abstimmung von Mensch und Pferd.',
   '[
     "Das Pferd aus dem Schritt an die am Boden liegende Stange heranführen und etwa 3 Meter davor das Tempo verlangsamen.",
     "Zunächst das Pferd vor der Stange halten lassen und es daraus schrittweise in die richtige Position führen; als erste Variante den ersten Schritt mit den Vorderbeinen über die Stange machen lassen und stehen bleiben.",
     "Als Ziel die Vorhand vor der Stange und die Hinterhand hinter der Stange stehen lassen.",
     "Nicht zu unsanft bremsen, denn das Pferd soll nicht dafür bestraft werden, dass es über die Stange geht; gelingt das Anhalten vor, über und nach der Stange nicht, zuerst das Halten üben."
   ]', 'main', 10, 'light', '{}', '{poles}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/,https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/}'),
-- https://horsesandpeople.com.au/10-easy-steps-to-easy-trailer-loading-rq/
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/
('00000000-0000-4000-8000-000000000934', NULL, 'groundwork', 'intermediate', '{vertrauen,gelassenheit}', 'Verladen in kleinen Schritten', 'Das Verladen wird in kleine Schritte zerlegt und ruhig aufgebaut, bis das Pferd gelassen in den Anhänger geht.',
   '[
     "Zuerst weg vom Anhänger ein Signal für vorwärts und eines für rückwärts erarbeiten: mit dem kleinsten Signal beginnen, eine Sekunde warten und es erst dann steigern; sobald das Pferd vortritt, das Signal beenden und loben.",
     "Dort beginnen, wo das Pferd sich wohlfühlt, und sich in Schritten dem Anhänger nähern.",
     "Am Fuß der Rampe zuerst einen Huf auf die Rampe stellen lassen, das Signal beenden und loben, dann einen Schritt zurück; in kleinen Schritten zu zwei, drei und vier Hufen steigern und die Pausen auf dem Anhänger nach und nach verlängern.",
     "Nie anbinden, bevor das Zurücktreten geübt wurde; zum Ausladen erst den Anbindeknoten lösen, dann die Verriegelung der Stange lösen lassen, die Stange herausnehmen, das Pferd gerade rückwärts herausführen und es seitlich an der Hängerklappe durch einen Helfer sichern lassen."
   ]', 'main', 20, 'light', '{}', '{trailer}', '{}', '{https://horsesandpeople.com.au/10-easy-steps-to-easy-trailer-loading-rq/,https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000935', NULL, 'groundwork', 'intermediate', '{aufmerksamkeit,mobilitaet,balance}', 'Arbeit am langen Seil', 'Am langen Seil lernt das Pferd, sich auch auf Distanz am Tempo des Menschen zu orientieren, und wird beweglicher.',
   '[
     "Das Pferd an einer kurzen Longe oder einem Bodenarbeitsseil in einem Abstand von mindestens 2,5 Metern, etwa einer Pferdelänge, bis 5 Metern arbeiten.",
     "Im Schritt und Trab auf einem Kreis sowie auf geraden und gebogenen Linien arbeiten; mit einer Volte von 6 bis 10 Metern in einer Ecke beginnen und die Größe an Gangart und Pferd anpassen.",
     "Exakte Übergänge zum Halten und fließende Übergänge zwischen Schritt und Trab zeigen; das Pferd orientiert sich auch bei größerer Distanz am Tempo der Führperson.",
     "Gerte oder Bodenarbeitsseil niemals im Kopfbereich einsetzen und den Kopf nie berühren."
   ]', 'main', 15, 'normal', '{}', '{lunge}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf}'),
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
-- https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/
('00000000-0000-4000-8000-000000000936', NULL, 'groundwork', 'advanced', '{seitengaenge,durchlaessigkeit,balance,mobilitaet}', 'Vorwärts-seitwärts übertreten', 'Gleichmäßig vorwärts-seitwärts übertreten gymnastiziert das Pferd und verbessert die Koordination.',
   '[
     "Das seitliche Verschieben und Folgen an der Bande oder auf einer vorgegebenen Linie üben.",
     "Das Pferd in einer gleichmäßigen Bewegung vorwärts-seitwärts übertreten lassen, zuerst im Schritt, später im Trab.",
     "Das erste Seitwärts am Boden auf einer kleinen Kreislinie erarbeiten und das Pferd wie den Zeiger einer Uhr um sich herum übertreten lassen; vor allem die Schultern seitwärts schicken und sich nicht auf die Hinterhand versteifen.",
     "Weicht das Pferd nicht und geht auf den Menschen zu, mit ausgestrecktem Arm stehen bleiben: So holt sich das Pferd beim Vortreten selbst einen Impuls am inneren Zügel und bleibt auf Distanz.",
     "Mit Pylonen den Wechsel zwischen vorwärts-seitwärts Weichen und vorwärts-seitwärts Folgen markieren."
   ]', 'main', 15, 'normal', '{}', '{cones}', '{}', '{https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf,https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/}'),

-- --- lunge ---
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
-- https://de.wikipedia.org/wiki/Longieren
('00000000-0000-4000-8000-000000000940', NULL, 'lunge', 'beginner', '{vertrauen,losgelassenheit,takt}', 'Longe: anführen und hinauslassen', 'Ruhig anführen und im Schritt auf den Zirkel hinauslassen: Das Pferd wärmt sich auf und gewinnt Vertrauen.',
   '[
     "Mit festem Schuhwerk und Handschuhen arbeiten und keine Sporen tragen, denn Sporen werden schnell zur Stolperfalle.",
     "Sich in Gehrichtung neben das Pferd stellen und es anführen; der Zirkel hat mindestens 12 Meter Durchmesser.",
     "Erfahrene Pferde Runde für Runde im Schritt aus dem kleinen Kreis um sich herum auf die große Zirkellinie hinausgehen lassen und die Longe dabei Schlaufe für Schlaufe über den Daumen ablassen.",
     "Junge und unerfahrene Pferde zunächst auf der großen Zirkellinie führen, bei richtigem Verhalten loben und erst die Position in der Zirkelmitte einnehmen, wenn sich das Pferd entspannt.",
     "Zum Aufwärmen etwa zehn Minuten im Schritt ohne Ausbinder longieren."
   ]', 'warmup', 10, 'light', '{}', '{lunge}', '{young}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/,https://de.wikipedia.org/wiki/Longieren}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000941', NULL, 'lunge', 'beginner', '{takt,rhythmus,durchlaessigkeit,losgelassenheit}', 'Tempounterschiede im Trab', 'Tempounterschiede innerhalb der Gangart fördern die Durchlässigkeit und aktivieren die Hinterhand.',
   '[
     "Erst wenn das Pferd kontrolliert in frischem Arbeitstempo mit ruhiger Frequenz traben kann und das Überschäumende weg ist, Tempounterschiede fordern.",
     "Zum Zulegen das Dreieck aus Longenführer, Longe und Pferd nach hinten verlagern, sodass das Pferd aus dem Zirkel herausgetrieben wird.",
     "Zum Zurücknehmen das Pferd mehr in den Zirkel führen und das Dreieck Richtung Pferdekopf verändern.",
     "Neigt das Pferd beim Zulegen dazu, schief zu werden, es ein paar Meter an der Bande geradeaus longieren und dann wieder einfangen.",
     "Beim Verändern der eigenen Position nicht zu sehr mit dem Pferd mitlaufen."
   ]', 'main', 10, 'normal', '{}', '{lunge}', '{}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000942', NULL, 'lunge', 'beginner', '{durchlaessigkeit,aufmerksamkeit,vertrauen}', 'Anhalten und Antreten auf Stimme', 'Ein eindeutiges Stimmsignal und feine Paraden machen das Pferd aufmerksam und durchlässig bis zum Halten.',
   '[
     "Aus dem Trab über den Schritt zum Halten durchparieren: Paraden und ein eindeutiges, gedehntes Stimmsignal wie „Schee-ritt“ geben.",
     "Genügt das nicht, das Dreieck in Richtung Pferdekopf verlagern; geht das Pferd gelassen im Schritt, mit Paraden und Stimme zum Halten auffordern und die Peitsche quer über oder unter der Longe vor das Pferd halten.",
     "Auf die Tonlage achten: Eine tiefe Stimme holt das Pferd zurück, eine höhere Stimme lässt es mehr aufpassen; auch ein beruhigendes „Brrr“ und ein dynamisches „Halt“ sind möglich.",
     "Ziel ist, dass das Pferd auf der Zirkellinie anhält und danach prompt wieder antrabt; bei richtigem Verhalten das Pferd loben."
   ]', 'main', 10, 'light', '{}', '{lunge}', '{}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000943', NULL, 'lunge', 'beginner', '{takt,losgelassenheit,anlehnung,mobilitaet,balance}', 'Vorwärts-abwärts longieren', 'Leichte Longenarbeit lehrt das Pferd, taktmäßig zu gehen, sich vorwärts-abwärts zu dehnen und den Rücken aufzuwölben.',
   '[
     "Das Pferd im Schritt etwa zehn Minuten ohne Ausbinder aufwärmen und danach in den drei Grundgangarten ausgebunden longieren.",
     "Auf fleißiges Vorwärts ohne Eilen achten: Das Pferd lernt, taktmäßig zu gehen, sich vorwärts-abwärts zu dehnen und den Rücken aufzuwölben; automatisch verbessert sich dann die Anlehnung.",
     "Viele Schritt-Trab- und Trab-Galopp-Übergänge fördern die Losgelassenheit.",
     "Den Ausbinder so verschnallen, dass die Stirn-Nasenlinie vor der Senkrechten bleibt; bei einem jungen Pferd etwas länger, damit es sich besser ausbalancieren kann.",
     "Zum Abschluss das Pferd zehn Minuten unausgebunden im Schritt entspannen lassen und nicht länger als 20 bis 30 Minuten longieren."
   ]', 'main', 20, 'light', '{}', '{lunge,side_reins}', '{young}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
('00000000-0000-4000-8000-000000000944', NULL, 'lunge', 'beginner', '{anlehnung,losgelassenheit,balance}', 'Hilfszügel richtig verschnallen', 'Ein richtig verschnallter Hilfszügel gibt dem Pferd Anlehnung und lässt es losgelassen über den Rücken schwingen.',
   '[
     "Einen Longiergurt über dem Sattel verschnallen oder einen Gurt mit viel Widerristfreiheit verwenden; einen Gurt nie direkt auf den Rücken legen.",
     "Jeden Ausbinder so verschnallen, dass die Stirn-Nasenlinie vor der Senkrechten ist; zur Kontrolle vor das Pferd stellen: Sind Nüsternrand und Buggelenk auf einer Höhe, ist das Pferd weder zu tief noch überstellt.",
     "Nicht benötigte Zügel eindrehen und am Kehlriemen fixieren, damit das Pferd nicht in einen lang herunterhängenden Zügel treten kann.",
     "Zuerst ohne Ausbinder im Schritt aufwärmen und danach ausbinden; ist der Übergang in den Schritt stockend, ist das Pferd oft zu eng ausgebunden.",
     "Wehrt sich das Pferd gegen einen starren Ausbinder, einen Gummi-Ausbinder nehmen, der bei Druck nachgibt, bis das Pferd Vertrauen in die Situation gewonnen hat."
   ]', 'main', 10, 'normal', '{}', '{lunge,side_reins}', '{}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/,https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
('00000000-0000-4000-8000-000000000945', NULL, 'lunge', 'beginner', '{anlehnung,vertrauen,losgelassenheit}', 'Longieren mit Kappzaum', 'Am Kappzaum bleibt das Pferdemaul unangetastet: Das hilft jungen und maulempfindlichen Pferden, Anlehnung zu finden.',
   '[
     "Prüfen, ob der Kappzaum passt: Der Metallbügel im gepolsterten Nasenriemen hält ihn stabil am Pferdekopf, und richtig und fest verschnallt rutscht er nicht ins Auge.",
     "Longe und Ausbinder im Kappzaum verschnallen: So bleibt das Pferdemaul völlig unangetastet; das eignet sich für junge Pferde und für Pferde, die empfindlich im Maul sind oder Angst vor der Reiterhand oder dem Ausbinder haben.",
     "Mit Longe, Peitsche, Stimme und der eigenen Position einwirken und die Peitsche ständig mitführen, auf Höhe des Sprunggelenks gehalten.",
     "Eine leichte Verbindung halten: Kleine Paraden stellen den Kontakt her, der Impuls bleibt nicht zu stark, denn auf Druck erfolgt Gegendruck."
   ]', 'main', 15, 'light', '{}', '{lunge,kappzaum,side_reins}', '{young}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/}'),
-- https://www.cavallo.de/reittraining/gymnastik-an-der-longe/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000946', NULL, 'lunge', 'intermediate', '{aufmerksamkeit,durchlaessigkeit,mobilitaet}', 'Fließender Handwechsel', 'Der fließende Handwechsel macht das Pferd aufmerksam und geschmeidig und verteilt die Belastung gleichmäßig.',
   '[
     "Das Pferd durch ein Stimmsignal aufmerksam machen und einladen, nach innen abzuwenden: mit entspannter Körperhaltung rückwärtsgehen und das Pferd zu sich bitten, die Longe bleibt dabei leicht gespannt.",
     "Beim einfachen Wechsel das Pferd anhalten, sobald es auf gerader Linie vor dir angekommen ist, per Stimmsignal oder indem die Hände mit gesenkter Peitsche ausgestreckt und angehoben werden.",
     "Die Peitsche unter der Longe hindurch wechseln und mit der anderen Hand greifen, die Vorhand in die neue Richtung weichen lassen und das Pferd wieder auf den Zirkel schicken; mehrmals auf beiden Seiten üben.",
     "Beim fließenden Wechsel nicht anhalten, sondern Longen- und Peitschenhand beim Rückwärtsgehen wechseln und das Pferd gleich in die neue Richtung wenden lassen; darauf achten, dass die Vorhand sauber und weit genug weicht.",
     "Als Faustregel alle fünf bis sieben Minuten die Richtung ändern."
   ]', 'main', 10, 'normal', '{00000000-0000-4000-8000-000000000842}', '{lunge}', '{}', '{https://www.cavallo.de/reittraining/gymnastik-an-der-longe/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000947', NULL, 'lunge', 'intermediate', '{durchlaessigkeit,rhythmus,balance}', 'Galoppsprünge verlängern und verkürzen', 'Verlängern und Verkürzen der Galoppsprünge an der Longe schult Durchlässigkeit und Rhythmus ohne Reitergewicht.',
   '[
     "Erst wenn das Pferd in der Gangart ausbalanciert, entspannt und im Takt arbeitet, den Wechsel der Tempi fordern.",
     "Im Arbeitsgalopp zwei Mal die Galoppsprünge verlängern und verkürzen, im Arbeitstrab zwei Mal die Trabtritte verlängern und verkürzen.",
     "Zum Verlängern das Dreieck nach hinten verlagern und das Pferd aus dem Zirkel heraustreiben, zum Verkürzen das Pferd mehr in den Zirkel führen und das Dreieck Richtung Pferdekopf verändern.",
     "Regelmäßig die Hand wechseln und nicht länger als 20 bis 30 Minuten longieren."
   ]', 'main', 10, 'normal', '{00000000-0000-4000-8000-000000000941}', '{lunge}', '{}', '{https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf,https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2023/
-- https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf
-- https://de.wikipedia.org/wiki/Longieren
('00000000-0000-4000-8000-000000000948', NULL, 'lunge', 'advanced', '{anlehnung,durchlaessigkeit,vertrauen}', 'Doppellonge: Gewöhnung in drei Schritten', 'Die Doppellonge erlaubt feinere Einwirkung als die einfache Longe: Das Pferd wird in drei Schritten daran gewöhnt.',
   '[
     "Eine Doppellonge von mindestens 18 Metern Länge aus griffigem, im vorderen Teil rundgenähtem Material und eine ausreichend lange Peitsche bereithalten; die Zügel aus der Trense ausschnallen und einen gepolsterten Longiergurt mit Widerristfreiheit verwenden.",
     "Erstens: Die innere Longe in den Gurtring einhängen und durch den inneren Trensenring in die Hand führen; die äußere Longe über den Rücken legen und durch den Gurtring zum Trensenring führen.",
     "Zweitens: Ist das Pferd entspannt und zufrieden, die äußere Longe um die Hinterhand legen; dabei Ruhe und Geduld haben, denn jedes Pferd reagiert anders auf die Longe am Hinterbein.",
     "Drittens: Akzeptiert das Pferd auf beiden Händen die Longe am Hinterbein, beide Longen in die Trensenringe einhängen und über den Gurt in die Hand führen; so sind Handwechsel ohne Umschnallen möglich.",
     "Das Pferd kann nur lernen, wenn es frei von Angst, Panik oder Zwang ist; für den Handwechsel durch den Zirkel wechseln und zu Beginn mit heftigen Reaktionen rechnen."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000944}', '{lunge,double_lunge}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2023/,https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf,https://de.wikipedia.org/wiki/Longieren}'),
-- https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/
('00000000-0000-4000-8000-000000000949', NULL, 'lunge', 'beginner', '{losgelassenheit,mobilitaet}', 'Ruhig ausgehen nach dem Longieren', 'Zum Abschluss unausgebunden im Schritt entspannen: Das Pferd kommt zur Ruhe und verarbeitet das Gelernte.',
   '[
     "Zum Abschluss das Pferd zehn Minuten unausgebunden im Schritt entspannen lassen.",
     "Die Schrittphase davor und danach geradeaus gehen lassen, vielleicht sogar an der frischen Luft.",
     "Das Pferd für ruhiges Mitgehen loben und danach in Ruhe entlassen."
   ]', 'cooldown', 10, 'light', '{}', '{lunge}', '{}', '{https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/,https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/}'),

-- --- hack ---
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
-- https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf
('00000000-0000-4000-8000-000000000950', NULL, 'hack', 'beginner', '{losgelassenheit,takt,mobilitaet}', 'Schritt zum Einstieg in den Ausritt', 'Zu Beginn mindestens zehn Minuten Schritt bringen den Organismus in Schwung und schützen vor Verletzungen.',
   '[
     "Anfänglich mindestens zehn Minuten im Schritt reiten, am besten mit hingegebenem Zügel, und die Anforderungen erst allmählich steigern.",
     "Mit der Hand der Nickbewegung folgen, damit das Pferd Kopf und Hals frei bewegen kann.",
     "Auf unebenem, hartem oder unübersichtlichem Boden nur Schritt reiten.",
     "Pferden mit Steifheit, etwa durch Arthrose, mehr Zeit zum Warmlaufen geben."
   ]', 'warmup', 10, 'light', '{}', '{}', '{young,senior}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/,https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2024/
('00000000-0000-4000-8000-000000000951', NULL, 'hack', 'beginner', '{losgelassenheit,takt,mobilitaet}', 'Aufwärmen auf dem Platz vor dem Ausritt', 'Auf dem Platz aufwärmen und Übergänge, Wendungen und leichten Sitz prüfen: So starten Pferd und Reiter sicher ins Gelände.',
   '[
     "Die Aufwärmphase kann alternativ auf dem Außenplatz durchgeführt werden, bevor es ins Gelände geht.",
     "Prüfen, ob das Pferd so sicher an den Hilfen steht, dass der Reiter jederzeit die Grundgangarten kontrollieren und Übergänge sowie Wendungen reiten kann.",
     "Den leichten Sitz mit kurzem Bügel in allen drei Gangarten ausbalancieren: Wer das sicher kann, ist bereit für den Ausritt.",
     "Die Hufe vor dem Ausritt gründlich kontrollieren und auskratzen."
   ]', 'warmup', 10, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/
('00000000-0000-4000-8000-000000000952', NULL, 'hack', 'beginner', '{losgelassenheit,mobilitaet}', 'Abkühlen im Schritt nach dem Ausritt', 'Zum Abschluss ruhig im Schritt am hingegebenen Zügel reiten: Das Pferd kühlt ab und entspannt.',
   '[
     "Abschließend mindestens zehn Minuten Schritt reiten und nach dem Galoppieren zuvor einen längeren Trababschnitt einplanen.",
     "Das lange Schrittreiten am langen oder hingegebenen Zügel fördert die Losgelassenheit.",
     "Nach dem Ausritt die Beine des Pferdes abkühlen.",
     "Die Hufe vor und nach jedem Ausritt gründlich kontrollieren und auskratzen."
   ]', 'cooldown', 10, 'light', '{}', '{}', '{young,senior}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/}'),
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
('00000000-0000-4000-8000-000000000953', NULL, 'hack', 'beginner', '{kondition,ausdauer,losgelassenheit,gelassenheit}', 'Ausritt im Schritt: Kondition und Losgelassenheit', 'Ausgiebiges Schrittreiten durch die Natur macht den Kopf frei, fördert die Losgelassenheit und legt die Grundlage für die Kondition.',
   '[
     "Die Strecke passend zu Ausbildungsstand und Fitnesslevel von Pferd und Reiter wählen und anfänglich wie abschließend mindestens zehn Minuten Schritt einplanen; immer mindestens zu zweit ausreiten.",
     "Ausgiebig Schritt reiten, am besten neben einem vertrauten Partner durch Wälder, Felder und Wiesen: Das fördert die Losgelassenheit und macht den Kopf frei.",
     "Längere Strecken als in der Reitbahn kommen der Kondition zugute; das lange Schrittreiten am langen oder hingegebenen Zügel fördert die Losgelassenheit.",
     "Wer an der Kondition arbeitet, braucht einen langfristigen Plan und ein Ziel; zwei bis dreimal pro Woche Konditionstraining in den Trainingsplan integrieren."
   ]', 'main', 30, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
('00000000-0000-4000-8000-000000000954', NULL, 'hack', 'beginner', '{kondition,kraft,balance,gelaende,mobilitaet,schwung}', 'Bergauf und bergab im Schritt', 'Hügel trainieren Muskeln, Bänder und Gelenke: Bergauf wächst die Schubkraft, bergab die Tragkraft.',
   '[
     "Hügel im Gelände nutzen: Bergauf reiten fördert die Schubkraft, bergab nimmt das Pferd Last auf und bekommt mehr Tragkraft.",
     "Bergauf braucht das Pferd mehr Kraft, um in gleichmäßigem Tempo voranzukommen; bergauf eher zulegen, bergab eher aufnehmen wollen, und die Belastung gut planen und dosieren.",
     "Im leichten Sitz mit kurzem Bügel reiten und mit längerem Zügel und offenem Ganaschenwinkel, damit das Pferd seinen Hals als Balancierstange nutzen kann.",
     "Bergauf- und Bergabreiten trainiert Muskeln, Bänder und Gelenke; Umfang und Intensität langsam steigern."
   ]', 'main', 20, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://blog.equisense.com/de/training-pferd-plan-ganze-saison/
('00000000-0000-4000-8000-000000000955', NULL, 'hack', 'beginner', '{kondition,ausdauer,takt,rhythmus}', 'Gleichmäßiger Trab über längere Strecken', 'Trabtraining im Arbeitstempo steigert die Grundkondition, wenn die Dauer langsam wächst.',
   '[
     "Je nach Ausgangslage zunächst nur über längere Strecken im Arbeitstempo traben und von anfänglich zehn Minuten schrittweise steigern.",
     "Dort, wo getrabt wird, sollte der Untergrund eben und federnd sein; vor und nach dem Traben ausreichend Schritt reiten.",
     "Takt und Tempo gleichmäßig halten; Ziel kann eine klassische Trabeinheit von 30 Minuten im flachen Gelände sein.",
     "Zwei bis dreimal pro Woche Konditionstraining einplanen; erst wenn das Trabtraining sicher klappt, erste Galoppreprisen einbauen."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000953}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://blog.equisense.com/de/training-pferd-plan-ganze-saison/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://alumni.bfh.ch/dam/jcr:7dc00f6f-7387-4071-9e88-38b1406834a0/burger-vielseitig-trainiert-brennpunkt-pferd-2019.pdf
('00000000-0000-4000-8000-000000000956', NULL, 'hack', 'intermediate', '{kondition,ausdauer,kraft,schwung,rhythmus}', 'Trab- und Galoppstrecken', 'Galoppreprisen im Gelände steigern Ausdauer und Rumpfmuskulatur, wenn der Umfang und nicht das Tempo wächst.',
   '[
     "In einem frischen Arbeitsgalopp beginnen und vor allem den Umfang steigern, nicht das Tempo; zweimal fünf Minuten mit einer Pause dazwischen kann das Ziel sein.",
     "Vor und nach dem Galoppieren einen längeren Trababschnitt einplanen und nur auf ebenem, federndem Untergrund galoppieren.",
     "Auf hartem Boden drohen Läsionen an Knochen, Gelenken und Hufen, auf tiefem Boden Muskelprobleme; den Boden bewusst wählen.",
     "Längere Galoppreprisen im Gelände mit wechselnden Bodenverhältnissen sorgen für mehr Ausdauer und kräftige Bauchmuskeln.",
     "Im leichten Sitz mit kurzem Bügel entlasten und das Pferd nicht auf die Vorhand fallen lassen, sondern vor dem Bein galoppieren."
   ]', 'main', 25, 'normal', '{00000000-0000-4000-8000-000000000955}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://alumni.bfh.ch/dam/jcr:7dc00f6f-7387-4071-9e88-38b1406834a0/burger-vielseitig-trainiert-brennpunkt-pferd-2019.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/
-- https://blog.equisense.com/de/training-pferd-plan-ganze-saison/
-- https://alumni.bfh.ch/dam/jcr:7dc00f6f-7387-4071-9e88-38b1406834a0/burger-vielseitig-trainiert-brennpunkt-pferd-2019.pdf
('00000000-0000-4000-8000-000000000957', NULL, 'hack', 'advanced', '{kondition,ausdauer,kraft,schwung}', 'Intervalle im Gelände', 'Intervalle mit Galopp und Erholung im Wechsel fordern Herz, Lunge und Muskulatur und gehören nur zu einem gut vorbereiteten Pferd.',
   '[
     "An einem Hügel im Galopp hochreiten und im Schritt wieder herunterreiten; Galoppieren am Berg ist ein Konditionstraining, vorausgesetzt es wird gut geplant und dosiert.",
     "Auf ebener Strecke zum Beispiel drei Minuten Galopp im Wechsel mit zwei Minuten Trab reiten und viermal wiederholen; je kürzer die Trabzeit, desto schwieriger.",
     "Galopptraining nur alle 5 bis 7 Tage ansetzen, in Abhängigkeit von Dauer und Intensität.",
     "Auf Boden, Beschlag und Schwachstellen des Pferdes achten und einmal pro Woche den Trab geradeaus und auf der Volte kontrollieren."
   ]', 'main', 25, 'demanding', '{00000000-0000-4000-8000-000000000956}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/,https://blog.equisense.com/de/training-pferd-plan-ganze-saison/,https://alumni.bfh.ch/dam/jcr:7dc00f6f-7387-4071-9e88-38b1406834a0/burger-vielseitig-trainiert-brennpunkt-pferd-2019.pdf}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
('00000000-0000-4000-8000-000000000958', NULL, 'hack', 'beginner', '{takt,losgelassenheit,gelaende}', 'Fleißig schreiten im Gelände', 'Takt, Fleiß und Raumgriff im Schritt zeigen die Losgelassenheit; draußen fördern wechselnde Böden das Gleichgewicht.',
   '[
     "Auf Takt, Fleiß und Raumgriff achten: Das sind die Kriterien für einen guten Schritt, und Losgelassenheit ist dafür die Voraussetzung.",
     "Mit der Hand der Nickbewegung folgen, damit das Pferd den Ganaschenwinkel öffnet und die Stirn-Nasenlinie vor die Senkrechte kommt.",
     "Den Schritt wechselseitig treiben: Wenn das rechte Hinterbein vorfußt, treibt der rechte Schenkel, links analog; aktiv aus dem Schenkel unterstützen bei weicher Hand und ruhigem Oberkörper.",
     "Bergauf, bergab, auf Sand oder Gras, auf tiefem oder trockenem Boden fordert das Gleichgewicht und fördert die Losgelassenheit; das kommt dem Schritt zugute.",
     "Planvoll und konzentriert schreiten, vom längeren zum kürzeren Zügel, auf geraden und gebogenen Linien."
   ]', 'main', 20, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/
('00000000-0000-4000-8000-000000000959', NULL, 'hack', 'beginner', '{schwung,durchlaessigkeit,rhythmus,takt,mobilitaet}', 'Tempowechsel auf dem Feldweg', 'Tempounterschiede und Übergänge auf dem Feldweg machen das Pferd geschmeidig und durchlässig.',
   '[
     "Auf dem Feldweg Übergänge aller Art einbauen, als Tempounterschiede innerhalb der Gangart oder von einer Gangart in die andere; das macht geschmeidig und durchlässig.",
     "Mit gefühlvoll herausgerittenen Tempounterschieden die Hinterbeine aktivieren; besonders beim Zurücknehmen nicht langsamer reiten, sondern die Dynamik durch das Treiben erhalten.",
     "Häufig zwischen Arbeitstrab und Arbeitsgalopp wechseln und den Galopp nur in kurzen Reprisen fordern.",
     "Das Tempo so wählen, dass das Pferd jederzeit sicher zu regulieren ist; in der Gruppe bestimmt das schwächste Pferd-Reiter-Paar das Tempo."
   ]', 'main', 15, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/
('00000000-0000-4000-8000-000000000960', NULL, 'hack', 'beginner', '{schwung,ausdauer,rhythmus,kondition,gelaende}', 'Frischer Galopp auf geeignetem Weg', 'Draußen galoppieren Pferde gern frisch vorwärts: auf ebenem, federndem Weg stärkt das den Rücken und die Rumpfmuskulatur.',
   '[
     "Im leichten Sitz mit kurzem Bügel galoppieren: Das entlastet und hält dem Pferd den Rücken frei; geschmeidig mitschwingen statt aktiv schieben.",
     "Nur auf ebenem, federndem Untergrund galoppieren; auf unebenem, hartem und unübersichtlichem Boden wird nur Schritt geritten.",
     "Den Galopp zunächst nur in kurzen Reprisen reiten und vor und nach dem Galoppieren einen längeren Trababschnitt einplanen.",
     "Im Gruppenritt nur dann schneller reiten, wenn alle damit einverstanden sind und die Pferde jederzeit sicher zu regulieren sind."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000955}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
('00000000-0000-4000-8000-000000000961', NULL, 'hack', 'beginner', '{losgelassenheit,durchlaessigkeit,seitengaenge,aufmerksamkeit}', 'Schenkelweichen im Gelände', 'Schenkelweichen auf dem Weg lenkt ein aufgeregtes Pferd auf die Hilfen und fördert Konzentration und Koordination.',
   '[
     "Ist das Pferd aufgeregt und abgelenkt, im Schritt schenkelweichen: Es fördert die Akzeptanz der Reiterhilfen und fordert Konzentration und Koordination, auch im Gelände.",
     "Mit leichter Stellung, aber ohne Biegung reiten: Der innere Schenkel treibt knapp hinter dem Gurt vorwärts-seitwärts, der äußere verwahrt gemeinsam mit dem äußeren Zügel.",
     "Das Vorwärts steht immer im Vordergrund; die Abstellung bleibt unter 45 Grad.",
     "Kommt das Pferd im Gelände nicht zum gelassenen Schreiten, fließend zwischen Schenkelweichen und Geradeausreiten wechseln."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
('00000000-0000-4000-8000-000000000962', NULL, 'hack', 'beginner', '{mobilitaet,balance,gelaende}', 'Auf allen Wegen: Böden, Wurzeln und Steine', 'Unterschiedliche Wege und Böden setzen zusätzliche Trainingsreize und machen das Pferd geschickter und trittsicherer.',
   '[
     "Beim Ausritt bewusst auf unterschiedlichen Wegen reiten: Das Pferd passt sich an das Geläuf an, wird geschickter und ausbalancierter.",
     "Auch unwegsameres Gelände nutzen: Über Wurzeln, Äste und Steine zu steigen verbessert Trittsicherheit, Koordination und Konzentration.",
     "Im Schritt mit hingegebenem Zügel reiten, damit das Pferd seine Kopf-Hals-Haltung selbst wählen und den Hals als Balancierstange nutzen kann.",
     "Auf hartem oder unübersichtlichem Boden nur Schritt reiten; auf gesunde Hufe achten."
   ]', 'main', 20, 'light', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
('00000000-0000-4000-8000-000000000963', NULL, 'hack', 'beginner', '{gelassenheit,vertrauen,gelaende}', 'Mit Führpferd: Wasser, Stufen und Gräben', 'Hinter einem ruhigen Führpferd lernt das Pferd Neues im Gelände mit Vertrauen kennen.',
   '[
     "Ein erfahrenes und ruhiges Führpferd nutzen: Vor allem Wasserdurchritte, Stufen oder Gräben sind dann für das Herdentier Pferd meistens kein Problem; später geht der Neuling vor.",
     "Nur mit wenigen anderen Pferden und kurzen Strecken beginnen und Umfang und Intensität langsam steigern.",
     "Alles, was das Pferd im Gelände lernt, soll es mit Vertrauen lernen; Ruhe, Geduld und Gefühl sind der Schlüssel, und erst wenn Neues ohne viel Aufwand gelingt, kommen neue Herausforderungen dazu.",
     "Das Pferd sofort loben, nicht erst am Ende der Einheit, und regelmäßig Schrittpausen mit hingegebenem Zügel einbauen."
   ]', 'main', 20, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/
('00000000-0000-4000-8000-000000000964', NULL, 'hack', 'beginner', '{gelaende,vertrauen,balance,springgymnastik}', 'Erste Baumstämme im Gelände', 'Niedrige Baumstämme vom langen, geraden Weg aus sind der beste Einstieg in feste Hindernisse im Gelände.',
   '[
     "Unterschiedliche, auf dem Boden liegende Baumstämme sind für den Einstieg am besten geeignet: Sie sind massiv und werden trotz geringer Höhe aufmerksam respektiert; viele kleine, an verschiedenen Stellen platzierte Baumstämme sind meist wertvoller als wenige aufwendige.",
     "Die ersten Hindernisse niedrig, einladend und mit sicherer Absprung- und Landestelle wählen und von einem langen, geraden Weg mit viel Platz dahinter anreiten.",
     "Alles, was später aus dem Galopp überwunden wird, zuerst aus dem Trab anreiten: So hat das Pferd mehr Zeit, sich einen Überblick zu verschaffen.",
     "Im Gelände mit längerem Zügel und offenem Ganaschenwinkel reiten; ein Halsriemen bietet Halt und verhindert das Festhalten am Zügel.",
     "Das Pferd sofort nach dem Sprung loben und Schrittpausen einbauen."
   ]', 'main', 20, 'normal', '{}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2024/
('00000000-0000-4000-8000-000000000965', NULL, 'hack', 'beginner', '{gelassenheit,vertrauen,aufmerksamkeit,gelaende}', 'Gruppenritt: Reihenfolge, Tempo und Handzeichen', 'Eine feste Reihenfolge, abgesprochene Handzeichen und das Tempo des schwächsten Paares machen den Gruppenritt sicher und ruhig.',
   '[
     "Vorab einen Anfangs- und einen Schlussreiter bestimmen: Der erste kennt die Strecke und gibt Richtung und Tempo vor, der letzte hält die Gruppe zusammen; vorne und hinten gehen gelassene Pferde mit erfahrenen Reitern.",
     "Das Tempo richtet sich nach dem schwächsten Pferd-Reiter-Paar; höheres Tempo nur, wenn alle einverstanden sind.",
     "Passende Abstände einhalten: keine zu großen Lücken, aber ein Sicherheitsabstand, damit Pferde nicht zum Aufholen angespornt werden und die Gefahr des Aufreitens klein bleibt.",
     "Gangartwechsel wie Anreiten, Antraben, Angaloppieren und Durchparieren vorher per Handzeichen ankündigen; die Zeichen vorab vereinbaren.",
     "Zu Beginn kleine Gruppen von zwei bis vier Pferden wählen, um das Pferd ans Ausreiten zu gewöhnen."
   ]', 'main', 30, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2024/}'),
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/
('00000000-0000-4000-8000-000000000966', NULL, 'hack', 'beginner', '{gelassenheit,vertrauen,aufmerksamkeit}', 'Im Straßenverkehr und bei Begegnungen', 'Wer das Pferd schon am Stall an Verkehr und Begegnungen gewöhnt, nimmt ihm die Furcht vor dem Unbekannten.',
   '[
     "Das Pferd schon auf dem Stallgrundstück mit solchen Situationen vertraut machen: Das nimmt ihm die Furcht vor dem Unbekannten.",
     "Reiter gelten als langsame Fahrzeuge: am rechten Rand reiten, die Verkehrsschilder beachten und Bürgersteige, Rad- und Fußwege meiden; Richtungswechsel per Handzeichen ankündigen.",
     "Begegnungen mit Menschen oder Tieren: durchparieren und mit ausreichend großem Sicherheitsabstand im Schritt vorbeireiten.",
     "Bevor eine Straße überquert wird, dicht aufschließen, prüfen, ob sie in beide Richtungen frei ist, und dann als geschlossene Gruppe zügig passieren."
   ]', 'main', 20, 'light', '{}', '{}', '{young}', '{https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/}'),
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/
('00000000-0000-4000-8000-000000000967', NULL, 'hack', 'intermediate', '{kraft,balance,mobilitaet,gelaende}', 'Klettern am Hang', 'Dosiertes Klettern stärkt Rücken- und Bauchmuskeln und macht das Pferd geschickt, geschmeidig und locker.',
   '[
     "Am Hang immer senkrecht zum Hang klettern und die Belastung vernünftig dosieren.",
     "Im leichten Sitz mit kurzem Bügel reiten, mit längerem Zügel und offenem Ganaschenwinkel, damit das Pferd seinen Hals als Balancierstange nutzen kann.",
     "Bergauf eher zulegen, bergab eher aufnehmen wollen."
   ]', 'main', 20, 'normal', '{00000000-0000-4000-8000-000000000954}', '{}', '{}', '{https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/,https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/,https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/}'),

-- --- western ---
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://www.wehorse.com/de/blog/westernreiten
-- https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf
('00000000-0000-4000-8000-000000000970', NULL, 'western', 'beginner', '{losgelassenheit,takt,mobilitaet}', 'Walk am losen Zügel zum Aufwärmen', 'Im Walk am losen Zügel aufwärmen: Das Pferd schreitet gelassen, locker und im Takt in die Arbeit.',
   '[
     "Zuerst im Walk am losen Zügel reiten: Der Walk ist eine natürliche Gangart im Viertakt, und das Pferd schreitet gelassen aus der Vorhand, während die Hinterhand unter den Schwerpunkt tritt.",
     "Zum Anreiten Beine und Steigbügel belasten, beide Schenkel gleichzeitig anstupfen und die Fäuste gleich weit vornehmen; als Stimmkommando „Go“.",
     "Im Westernreiten wird die Hilfe zum Gangartwechsel nur einmal gegeben, und das Pferd behält die Gangart bei, bis es eine andere Hilfe bekommt; Hilfen sind Impulse, kein dauernder Druck.",
     "Pferden mit Steifheit, etwa durch Arthrose, mehr Zeit zum Warmlaufen geben."
   ]', 'warmup', 10, 'light', '{}', '{}', '{young,senior}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://www.wehorse.com/de/blog/westernreiten,https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://www.wehorse.com/de/blog/westernreiten
('00000000-0000-4000-8000-000000000971', NULL, 'western', 'beginner', '{losgelassenheit,takt,anlehnung,mobilitaet}', 'Jog vorwärts-abwärts', 'Im Jog mit Dehnungsbereitschaft lösen: Rhythmus und Nachgiebigkeit sind die Basis für alle Manöver.',
   '[
     "Aus dem Walk antraben: gleiches Vorgehen wie beim Anreiten, als Stimmkommando Schnalzen.",
     "Im Jog auf Rhythmus achten: Der Jog ist eine Gangart im Zweitakt, die Bewegungen sind rhythmisch, weich und leicht federnd.",
     "Die Nachgiebigkeit prüfen: Das Pferd folgt den Hilfen und nimmt willig die Dehnungshaltung an; die Oberlinie bleibt ein positiver Spannungsbogen mit dem Kopf an oder vor der Senkrechten.",
     "Schenkelhilfen ausschließlich am oder hinter dem Gurt einsetzen und das Pferd mit feinen Hilfen führen."
   ]', 'warmup', 10, 'light', '{}', '{}', '{young}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://www.wehorse.com/de/blog/westernreiten}'),
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/
-- https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/
('00000000-0000-4000-8000-000000000972', NULL, 'western', 'beginner', '{losgelassenheit,mobilitaet}', 'Cool-down im Walk am langen Zügel', 'Zum Abschluss zurück in den Walk und am langen Zügel ausgehen: Das Pferd entspannt und verarbeitet das Gelernte.',
   '[
     "In die niedrigere Gangart wechseln: Gewicht nach hinten verlagern, beide Fäuste gleich weit zurücknehmen und gleichzeitig etwas treiben, damit das Pferd nicht ganz stehen bleibt; als Stimmkommando „Easy“.",
     "Abschließend mindestens zehn Minuten im Walk am langen Zügel reiten, wenn möglich draußen.",
     "Nach einem intensiven Training folgt am besten eine Aktivpause wie leichtes Longieren oder eine Passivpause wie Weidegang."
   ]', 'cooldown', 10, 'light', '{}', '{}', '{senior}', '{https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/,https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/}'),
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal
('00000000-0000-4000-8000-000000000973', NULL, 'western', 'beginner', '{durchlaessigkeit,balance,aufmerksamkeit}', 'Stop und Verharren', 'Ein ausbalancierter Stop aus allen Gangarten und ruhiges Verharren zeigen, dass das Pferd auf feine Signale hört.',
   '[
     "Zum Anhalten das Gewicht nach hinten verlagern, die Beine vom Pferdekörper nehmen und beide Fäuste ein kleines Stück zurücknehmen; als Stimmkommando „Whoa“.",
     "Sobald das Pferd angehalten hat, gleich wieder die Hände nach vorne nehmen: Das ist die Belohnung.",
     "Der Stop ist ein ausbalanciertes Anhalten: Das Pferd bleibt gerade, nimmt mit der Hinterhand mehr Last auf, und die Vorhand bleibt leicht.",
     "Beim Verharren das Pferd entspannt und regungslos stehen lassen; für einen Moment kommen alle vier Beine zum Stillstand.",
     "Mit Schritt-Halt-Schritt-Halt beginnen und in kleinen Reprisen über einen längeren Zeitraum arbeiten."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal}'),
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.thegentletouch.de/post/r%C3%BCckw%C3%A4rtsrichten-mit-leichtigkeit
-- https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/
('00000000-0000-4000-8000-000000000974', NULL, 'western', 'beginner', '{durchlaessigkeit,balance,aufmerksamkeit}', 'Back up aus dem Halt', 'Das Rückwärts lehrt das Pferd, von vorwärts auf rückwärts umzudenken, und kräftigt Bauch, Rumpf und Hinterhand.',
   '[
     "Das Rückwärts zuerst am Boden vorbereiten und erst dann unter dem Reiter üben; das Pferd lernt dabei, bei Tempoverlangsamungen und beim Anhalten aus dem Vorwärts auf rückwärts umzudenken.",
     "Zum Rückwärtsrichten den Pferderücken entlasten, indem das Gewicht mehr auf die Beine verlagert wird; weit hinten im Sattel einsetzen und mit beiden Beinen am Gurt treiben; als Stimmkommando „Back“.",
     "Geht das Pferd nicht sofort rückwärts, und nur dann, leicht an den Zügeln zupfen.",
     "Das Back up ist eine Rückwärtsbewegung im Zweitakt mit diagonaler Fußfolge; der saubere Zweitakt ist zu Beginn noch nicht zwingend, der Ablauf ruhig und ohne Hektik.",
     "Beim ersten Schritt rückwärts die Aufforderung beenden und das Pferd loben."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.thegentletouch.de/post/r%C3%BCckw%C3%A4rtsrichten-mit-leichtigkeit,https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal
-- https://www.kreinberg.college/lp/westerndressage
('00000000-0000-4000-8000-000000000975', NULL, 'western', 'beginner', '{seitengaenge,durchlaessigkeit,aufmerksamkeit,mobilitaet}', 'Vorhandwendung', 'Die Hinterhand wandert um die Vorhand: Das gymnastiziert das Pferd und schult die Reaktion auf den Schenkel.',
   '[
     "Bei der Vorhandwendung fußt die Vorhand auf einer Stelle auf und ab; die Hinterbeine bewegen sich kreisförmig um die Vorhand und sollen dabei voreinander kreuzen.",
     "Das Pferd ist gerade gerichtet oder leicht gegen die Bewegungsrichtung gestellt.",
     "Bei einer Vorhandwendung rechts bewegt das Pferd seine Hinterbeine nach links, bei einer Vorhandwendung links nach rechts.",
     "Schenkelhilfen ausschließlich am oder hinter dem Gurt einsetzen.",
     "Mit Schritt-Halt-Schritt-Halt beginnen und die Hilfen mit jeder Wiederholung etwas feiner, weicher und im besseren Timing geben."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal,https://www.kreinberg.college/lp/westerndressage}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.wehorse.com/de/blog/westernreiten
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
('00000000-0000-4000-8000-000000000976', NULL, 'western', 'beginner', '{takt,rhythmus,losgelassenheit,biegung,durchlaessigkeit}', 'Zirkel mit Speed Control', 'Zirkel in verschiedenen Größen und Tempi lehren das Pferd, das Tempo fein regulieren zu lassen.',
   '[
     "Zirkel nach Vorgabe von Gangart, Größe und Geschwindigkeit reiten.",
     "Zirkel werden meist groß und schnell oder klein und langsam geritten; wichtig ist, dass der Reiter das Tempo jederzeit fein regulieren kann.",
     "Zum Biegen in Volte oder Zirkel das Gewicht leicht in die gewünschte Richtung verlagern und die innere Faust leicht anheben; der äußere Schenkel liegt je nach Enge der Kurve stärker oder schwächer an, der innere Schenkel wird etwas vom Pferdekörper genommen.",
     "Volten sollten für Reiter der LK 5 und 4 nicht weniger als 12 Meter Durchmesser haben, für LK 1 bis 3 nicht weniger als 10 Meter."
   ]', 'main', 15, 'normal', '{}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.wehorse.com/de/blog/westernreiten,https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html}'),
-- https://www.wehorse.com/de/blog/westernreiten
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
('00000000-0000-4000-8000-000000000977', NULL, 'western', 'beginner', '{anlehnung,biegung,durchlaessigkeit}', 'Neck Reining: Richtungswechsel', 'Einhändig am Hals geführt wechselt das Pferd die Richtung auf feine Zügelhilfen.',
   '[
     "Das Grundprinzip des einhändigen Reitens, des Neck Reining: Immer ein Zügel liegt am Hals an, der andere wird seitlich vom Hals weg geöffnet, um die Richtung zu bestimmen.",
     "Zum Richtungswechsel das Gewicht ganz leicht in die gewünschte Richtung verlagern; beide Fäuste bleiben auf gleicher Höhe, der innere Zügel ist weit weg vom Pferdehals, der äußere liegt am Hals an.",
     "Das innere Bein liegt am Gurt, das äußere eine Handbreit hinter dem Gurt.",
     "Hilfen so fein wie möglich geben; feine Hilfen sind möglich, wenn der Reiter sein Pferd nicht stört, nicht behindert und nicht irritiert."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://www.wehorse.com/de/blog/westernreiten,https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
('00000000-0000-4000-8000-000000000978', NULL, 'western', 'beginner', '{seitengaenge,durchlaessigkeit,geraderichten}', 'Leg-Yield im Walk', 'Das Pferd weicht parallel vorwärts-seitwärts und lernt, auf den seitwärtstreibenden Schenkel zu reagieren.',
   '[
     "Beim Leg-Yield bewegt sich das Pferd parallel vorwärts-seitwärts, bleibt dabei gerade und darf leicht gegen die Bewegungsrichtung gestellt sein; Vorder- und Hinterbeine überkreuzen.",
     "Schenkelhilfen ausschließlich am oder hinter dem Gurt einsetzen; das Vorwärts steht immer im Vordergrund.",
     "Zu Beginn ist es einfacher, den der Bande zugewandten Schenkel weichen zu lassen: Die Bande begrenzt, und der Reiter kann sich auf das Vorwärts-Seitwärtstreiben konzentrieren.",
     "Die Abstellung bleibt unter 45 Grad, damit Takt und Fleiß erhalten bleiben."
   ]', 'main', 10, 'light', '{}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/}'),
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.wehorse.com/de/blog/westernreiten
('00000000-0000-4000-8000-000000000979', NULL, 'western', 'beginner', '{durchlaessigkeit,takt,rhythmus}', 'Übergänge Walk, Jog und Lope', 'Klare Übergänge mit nur einer Hilfe zeigen, dass das Pferd durchlässig ist und die Gangart beibehält.',
   '[
     "Von Schritt zu Trab wie beim Anreiten vorgehen und als Stimmkommando Schnalzen geben.",
     "Von Trab zu Galopp das äußere Bein eine Handbreit hinter den Gurt anlegen und anstupfen, bis das Pferd galoppiert; danach den Schenkel nur sanft anliegen lassen; als Stimmkommando das Kussgeräusch.",
     "Der Lope ist eine Gangart im Dreitakt: Auf der linken Hand zeigt das Pferd den Linksgalopp, auf der rechten den Rechtsgalopp; die Bewegungen sind rhythmisch, weich und leicht federnd.",
     "Zum Verlangsamen das Gewicht nach hinten verlagern, beide Fäuste gleich weit zurücknehmen und etwas treiben; als Stimmkommando „Easy“.",
     "Die Hilfe zum Gangartwechsel nur einmal geben: Das Pferd behält die neue Gangart bei, bis es eine andere Hilfe bekommt."
   ]', 'main', 15, 'normal', '{}', '{}', '{}', '{https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.wehorse.com/de/blog/westernreiten}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
('00000000-0000-4000-8000-000000000980', NULL, 'western', 'beginner', '{takt,rhythmus,balance,mobilitaet}', 'Stangenarbeit: Walk over und Jog over', 'Stangen im Trail-Abstand fördern Takt, Balance und Aufmerksamkeit beim Überschreiten.',
   '[
     "Für den Walk over die Stangen im Abstand von 40 bis 60 Zentimetern legen, die Erhöhung beträgt höchstens 30 Zentimeter; bei Erhöhungen sollte der Mindestabstand 55 Zentimeter betragen.",
     "Für den Jog over einen Abstand von 90 bis 105 Zentimetern wählen, die Erhöhung beträgt höchstens 20 Zentimeter.",
     "Alle erhöhten Teile so sichern, dass sie nicht rollen können.",
     "Zwischen den Hindernissen eine Trabstrecke von mindestens 10 Metern und eine Galoppstrecke von mindestens 20 Metern lassen, um die Gangqualität beurteilen zu können.",
     "Im Schritt mit hingegebenem Zügel das Pferd seine Kopf-Hals-Haltung selbst wählen lassen, damit es den Hals als Balancierstange nutzen kann."
   ]', 'main', 15, 'normal', '{}', '{poles}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/
-- https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf
('00000000-0000-4000-8000-000000000981', NULL, 'western', 'beginner', '{vertrauen,aufmerksamkeit,gelassenheit}', 'Trail: Tor öffnen, durchreiten, schließen', 'Am Tor üben Pferd und Reiter, aufmerksam und gelassen Aufgaben des Trail zu lösen.',
   '[
     "Ein Tor mit einer Öffnung von mindestens 250 Zentimetern aufbauen; die Aufgabe heißt Öffnen, Durchreiten und Schließen.",
     "Das Pferd soll die Hindernisse aufmerksam, in angemessener Geschwindigkeit, korrekt und flüssig bewältigen und den Reiterhilfen willig folgen.",
     "Das Pferd zunächst geführt mit den Aufgaben vertraut machen, denn Trail-Aufgaben sind für das Pferd anspruchsvoll.",
     "Bei Schwierigkeiten den Schwierigkeitsgrad verringern und die Übung mit einer gelungenen Ausführung beenden."
   ]', 'main', 15, 'light', '{}', '{obstacles}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/,https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
('00000000-0000-4000-8000-000000000982', NULL, 'western', 'beginner', '{vertrauen,aufmerksamkeit,gelassenheit,durchlaessigkeit}', 'Trail: Slalom und Back up durch die Gasse', 'Mit Pylonen lassen sich Slalom und Rückwärts durch eine Gasse ohne Stangen üben.',
   '[
     "Einen Slalom im Walk mit einem Abstand von mindestens 200 Zentimetern aufbauen, im Jog mit mindestens 250 Zentimetern.",
     "Für das Back up eine Gasse aus mindestens 3 Pylonen oder Stangen mit einem Abstand von mindestens 100 Zentimetern bilden.",
     "Rückwärtsreiten über Gegenstände oder Hindernisse ist im Trail verboten.",
     "Aufmerksam, flüssig und mit balancierten, weichen Übergängen reiten; Vorwegnehmen der Aufgabe und zögerliches Heranreiten sind Mängel."
   ]', 'main', 15, 'normal', '{}', '{cones}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal
('00000000-0000-4000-8000-000000000983', NULL, 'western', 'intermediate', '{seitengaenge,durchlaessigkeit,balance}', 'Hinterhandwendung', 'Die Vorhand wandert um das innere Hinterbein: Das Pferd lernt, Gewicht auf die Hinterhand zu bringen.',
   '[
     "Die Hinterhandwendung ist eine Drehung bis zu 360 Grad um das innere Hinterbein, wobei die Vorderbeine vorwärts-seitwärts kreuzen.",
     "Mit Schritt-Halt-Schritt-Halt beginnen und die Drehung in kleinen Reprisen über einen längeren Zeitraum üben.",
     "Das Rückwärtsrichten, die Hinterhandwendung und die Schaukel sind gymnastizierende Übungen, die dem Pferd das Umdenken von vorwärts auf rückwärts vermitteln.",
     "Ziel ist eine leichte, ruhige und dynamisch-flüssige Ausführung."
   ]', 'main', 10, 'normal', '{00000000-0000-4000-8000-000000000975,00000000-0000-4000-8000-000000000974}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal
-- https://www.kreinberg.college/lp/westerndressage
('00000000-0000-4000-8000-000000000984', NULL, 'western', 'intermediate', '{seitengaenge,durchlaessigkeit,geraderichten}', 'Sidepass', 'Der Sidepass übt das Seitwärtsweichen mit überkreuzenden Vorder- und Hinterbeinen.',
   '[
     "Beim Sidepass kreuzen Vorder- und Hinterbeine abwechselnd voreinander; das Pferd bleibt gerade oder in Bewegungsrichtung gestellt und bewegt sich seitwärts in die vorgeschriebene Richtung.",
     "Schenkelhilfen ausschließlich am oder hinter dem Gurt einsetzen.",
     "Mit Schritt-Halt-Schritt-Halt beginnen und nur wenige Tritte seitwärts verlangen, dann steigern.",
     "Die Hilfen mit jeder Wiederholung etwas feiner und weicher geben."
   ]', 'main', 10, 'normal', '{00000000-0000-4000-8000-000000000978}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal,https://www.kreinberg.college/lp/westerndressage}'),
-- https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.wehorse.com/de/blog/westernreiten
('00000000-0000-4000-8000-000000000985', NULL, 'western', 'intermediate', '{schwung,takt,rhythmus,durchlaessigkeit}', 'Lope und Zirkel mit Tempowechsel', 'Im Lope Tempo und Rhythmus fein regulieren: Die Basis für Rundown und Stop.',
   '[
     "Aus dem Jog angaloppieren, das äußere Bein eine Handbreit hinter den Gurt anlegen und anstupfen, bis das Pferd galoppiert; danach den Schenkel nur sanft anliegen lassen.",
     "Zirkel im Lope nach Vorgabe von Größe und Geschwindigkeit reiten: groß und schnell oder klein und langsam.",
     "Das Tempo jederzeit fein regulieren und den Lope als Gangart im Dreitakt rhythmisch, weich und leicht federnd halten.",
     "Beim Run Down einen gleichmäßigen und kontrollierten Tempoaufbau im Lope zeigen."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000979,00000000-0000-4000-8000-000000000976}', '{}', '{}', '{https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html,https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.wehorse.com/de/blog/westernreiten}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
('00000000-0000-4000-8000-000000000986', NULL, 'western', 'intermediate', '{takt,rhythmus,balance,schwung}', 'Trail: Lope over', 'Stangen im Lope-Abstand fordern Takt, Aufmerksamkeit und Balance im Galopp.',
   '[
     "Für das Lope over die Stangen im Abstand von 180 bis 210 Zentimetern legen, die Erhöhung beträgt höchstens 20 Zentimeter.",
     "Das Pferd soll beim Lope over keine Stange zwischen die Vorder- oder Hinterhufe nehmen und keinen Tritt in einen Zwischenraum auslassen.",
     "Alle erhöhten Teile so sichern, dass sie nicht rollen können; zwischen den Hindernissen eine Galoppstrecke von mindestens 20 Metern lassen.",
     "Mit dem Walk over und dem Jog over beginnen und erst dann den Lope over reiten."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000980,00000000-0000-4000-8000-000000000979}', '{poles}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.wehorse.com/de/blog/westernreiten
-- https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal
('00000000-0000-4000-8000-000000000987', NULL, 'western', 'advanced', '{seitengaenge,durchlaessigkeit,balance,kraft}', 'Spin', 'Der Spin ist eine schnelle Drehung auf der Hinterhand und verlangt Kraft, Balance und feine Hilfen.',
   '[
     "Der Spin ist eine 360-Grad-Drehung um ein stationäres inneres Hinterbein; das äußere Hinterbein und die beiden Vorderbeine sorgen für den Antrieb, und die Vorderbeine kreuzen voreinander.",
     "Beim Spin verlagert das Pferd sein Gewicht auf das innere Hinterbein, das während der Drehung fest am Platz stehen bleibt.",
     "Mit der Hinterhandwendung beginnen und Drehungen in kleinen Reprisen über einen längeren Zeitraum üben, denn die Muskulatur braucht Zeit.",
     "Die Übung nur mit einem Pferd üben, das die Hinterhandwendung sicher beherrscht."
   ]', 'main', 10, 'demanding', '{00000000-0000-4000-8000-000000000983}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.wehorse.com/de/blog/westernreiten,https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/
('00000000-0000-4000-8000-000000000988', NULL, 'western', 'advanced', '{seitengaenge,versammlung,durchlaessigkeit,geraderichten}', 'Two Track', 'Diagonal vorwärts-seitwärts verbessert Durchlässigkeit und Gymnastizierung und bereitet die Versammlung vor.',
   '[
     "Beim Two Track bewegt sich das Pferd diagonal vorwärts-seitwärts und ist dabei gerade oder in Bewegungsrichtung gestellt und gebogen.",
     "Das Pferd soll bei allen Seitwärtsbewegungen mit Vorder- und Hinterbeinen überkreuzen.",
     "Schenkelhilfen ausschließlich am oder hinter dem Gurt einsetzen.",
     "Mit dem Leg-Yield beginnen und das Two Track erst reiten, wenn das Pferd parallel vorwärts-seitwärts sicher weicht."
   ]', 'main', 15, 'normal', '{00000000-0000-4000-8000-000000000978,00000000-0000-4000-8000-000000000984}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/}'),
-- https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf
-- https://www.wehorse.com/de/blog/westernreiten
('00000000-0000-4000-8000-000000000989', NULL, 'western', 'advanced', '{durchlaessigkeit,balance,geraderichten}', 'Galoppwechsel', 'Der Galoppwechsel verlangt ein gerades, ausbalanciertes und durchlässiges Pferd.',
   '[
     "Ein Galoppwechsel ist der Wechsel vom Links- in den Rechtsgalopp oder umgekehrt; beim einfachen Galoppwechsel wechselt das Pferd den Galopp über Schritt oder Trab.",
     "Beim fliegenden Galoppwechsel wechselt das Pferd ohne Gangartunterbrechung innerhalb eines Galoppsprunges; Vorder- und Hinterbeine wechseln im gleichen Galoppsprung.",
     "Den Wechsel auf den Punkt genau an der vorgesehenen Stelle ausführen und dabei ein gleichbleibendes Tempo halten.",
     "Mit einfachen Wechseln beginnen und den fliegenden Wechsel erst üben, wenn das Pferd die Gangart sicher beibehält und gerade bleibt."
   ]', 'main', 15, 'demanding', '{00000000-0000-4000-8000-000000000985}', '{}', '{}', '{https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf,https://www.wehorse.com/de/blog/westernreiten}')
ON CONFLICT (id) DO UPDATE SET
    discipline = EXCLUDED.discipline, level = EXCLUDED.level, goal_tags = EXCLUDED.goal_tags,
    title = EXCLUDED.title, summary = EXCLUDED.summary, steps = EXCLUDED.steps, phase = EXCLUDED.phase,
    minutes = EXCLUDED.minutes, intensity = EXCLUDED.intensity, requires = EXCLUDED.requires,
    equipment = EXCLUDED.equipment, suits = EXCLUDED.suits, sources = EXCLUDED.sources
WHERE exercises.stable_id IS NULL;

-- Tags of existing rows that the coverage rules of the spec need (the tag lists stay unique and sorted).
-- 834 "Vorhand weichen lassen" is a lateral exercise on the ground and mobilises the horse.
UPDATE exercises SET goal_tags = ARRAY(SELECT DISTINCT t FROM unnest(goal_tags || '{seitengaenge,mobilitaet}'::text[]) AS t ORDER BY t)
WHERE id = '00000000-0000-4000-8000-000000000834' AND stable_id IS NULL;

-- Progressions (follow-up exercise), set once all rows exist. Chains stay inside a library; a chain may end at
-- a row of 0250 or 0280.
UPDATE exercises e SET next_exercise_id = p.next
FROM (VALUES
    ('00000000-0000-4000-8000-000000000850'::uuid, '00000000-0000-4000-8000-000000000851'::uuid),
    ('00000000-0000-4000-8000-000000000851'::uuid, '00000000-0000-4000-8000-000000000852'::uuid),
    ('00000000-0000-4000-8000-000000000852'::uuid, '00000000-0000-4000-8000-000000000864'::uuid),
    ('00000000-0000-4000-8000-000000000854'::uuid, '00000000-0000-4000-8000-000000000858'::uuid),
    ('00000000-0000-4000-8000-000000000855'::uuid, '00000000-0000-4000-8000-000000000857'::uuid),
    ('00000000-0000-4000-8000-000000000856'::uuid, '00000000-0000-4000-8000-000000000871'::uuid),
    ('00000000-0000-4000-8000-000000000857'::uuid, '00000000-0000-4000-8000-000000000856'::uuid),
    ('00000000-0000-4000-8000-000000000858'::uuid, '00000000-0000-4000-8000-000000000866'::uuid),
    ('00000000-0000-4000-8000-000000000859'::uuid, '00000000-0000-4000-8000-000000000860'::uuid),
    ('00000000-0000-4000-8000-000000000861'::uuid, '00000000-0000-4000-8000-000000000859'::uuid),
    ('00000000-0000-4000-8000-000000000864'::uuid, '00000000-0000-4000-8000-000000000884'::uuid),
    ('00000000-0000-4000-8000-000000000865'::uuid, '00000000-0000-4000-8000-000000000867'::uuid),
    ('00000000-0000-4000-8000-000000000866'::uuid, '00000000-0000-4000-8000-000000000867'::uuid),
    ('00000000-0000-4000-8000-000000000867'::uuid, '00000000-0000-4000-8000-000000000810'::uuid),
    ('00000000-0000-4000-8000-000000000868'::uuid, '00000000-0000-4000-8000-000000000869'::uuid),
    ('00000000-0000-4000-8000-000000000872'::uuid, '00000000-0000-4000-8000-000000000870'::uuid),
    ('00000000-0000-4000-8000-000000000873'::uuid, '00000000-0000-4000-8000-000000000874'::uuid),
    ('00000000-0000-4000-8000-000000000874'::uuid, '00000000-0000-4000-8000-000000000875'::uuid),
    ('00000000-0000-4000-8000-000000000876'::uuid, '00000000-0000-4000-8000-000000000877'::uuid),
    ('00000000-0000-4000-8000-000000000877'::uuid, '00000000-0000-4000-8000-000000000878'::uuid),
    ('00000000-0000-4000-8000-000000000878'::uuid, '00000000-0000-4000-8000-000000000879'::uuid),
    ('00000000-0000-4000-8000-000000000879'::uuid, '00000000-0000-4000-8000-000000000883'::uuid),
    ('00000000-0000-4000-8000-000000000880'::uuid, '00000000-0000-4000-8000-000000000876'::uuid),
    ('00000000-0000-4000-8000-000000000882'::uuid, '00000000-0000-4000-8000-000000000881'::uuid),
    ('00000000-0000-4000-8000-000000000890'::uuid, '00000000-0000-4000-8000-000000000820'::uuid),
    ('00000000-0000-4000-8000-000000000906'::uuid, '00000000-0000-4000-8000-000000000826'::uuid),
    ('00000000-0000-4000-8000-000000000907'::uuid, '00000000-0000-4000-8000-000000000909'::uuid),
    ('00000000-0000-4000-8000-000000000908'::uuid, '00000000-0000-4000-8000-000000000911'::uuid),
    ('00000000-0000-4000-8000-000000000909'::uuid, '00000000-0000-4000-8000-000000000910'::uuid),
    ('00000000-0000-4000-8000-000000000911'::uuid, '00000000-0000-4000-8000-000000000912'::uuid),
    ('00000000-0000-4000-8000-000000000912'::uuid, '00000000-0000-4000-8000-000000000913'::uuid),
    ('00000000-0000-4000-8000-000000000913'::uuid, '00000000-0000-4000-8000-000000000914'::uuid),
    ('00000000-0000-4000-8000-000000000914'::uuid, '00000000-0000-4000-8000-000000000918'::uuid),
    ('00000000-0000-4000-8000-000000000918'::uuid, '00000000-0000-4000-8000-000000000919'::uuid),
    ('00000000-0000-4000-8000-000000000920'::uuid, '00000000-0000-4000-8000-000000000924'::uuid),
    ('00000000-0000-4000-8000-000000000923'::uuid, '00000000-0000-4000-8000-000000000924'::uuid),
    ('00000000-0000-4000-8000-000000000924'::uuid, '00000000-0000-4000-8000-000000000925'::uuid),
    ('00000000-0000-4000-8000-000000000925'::uuid, '00000000-0000-4000-8000-000000000926'::uuid),
    ('00000000-0000-4000-8000-000000000926'::uuid, '00000000-0000-4000-8000-000000000835'::uuid),
    ('00000000-0000-4000-8000-000000000927'::uuid, '00000000-0000-4000-8000-000000000835'::uuid),
    ('00000000-0000-4000-8000-000000000928'::uuid, '00000000-0000-4000-8000-000000000929'::uuid),
    ('00000000-0000-4000-8000-000000000929'::uuid, '00000000-0000-4000-8000-000000000930'::uuid),
    ('00000000-0000-4000-8000-000000000930'::uuid, '00000000-0000-4000-8000-000000000931'::uuid),
    ('00000000-0000-4000-8000-000000000932'::uuid, '00000000-0000-4000-8000-000000000933'::uuid),
    ('00000000-0000-4000-8000-000000000940'::uuid, '00000000-0000-4000-8000-000000000941'::uuid),
    ('00000000-0000-4000-8000-000000000941'::uuid, '00000000-0000-4000-8000-000000000942'::uuid),
    ('00000000-0000-4000-8000-000000000942'::uuid, '00000000-0000-4000-8000-000000000943'::uuid),
    ('00000000-0000-4000-8000-000000000943'::uuid, '00000000-0000-4000-8000-000000000944'::uuid),
    ('00000000-0000-4000-8000-000000000944'::uuid, '00000000-0000-4000-8000-000000000845'::uuid),
    ('00000000-0000-4000-8000-000000000953'::uuid, '00000000-0000-4000-8000-000000000954'::uuid),
    ('00000000-0000-4000-8000-000000000954'::uuid, '00000000-0000-4000-8000-000000000955'::uuid),
    ('00000000-0000-4000-8000-000000000955'::uuid, '00000000-0000-4000-8000-000000000956'::uuid),
    ('00000000-0000-4000-8000-000000000956'::uuid, '00000000-0000-4000-8000-000000000957'::uuid),
    ('00000000-0000-4000-8000-000000000958'::uuid, '00000000-0000-4000-8000-000000000959'::uuid),
    ('00000000-0000-4000-8000-000000000959'::uuid, '00000000-0000-4000-8000-000000000960'::uuid),
    ('00000000-0000-4000-8000-000000000963'::uuid, '00000000-0000-4000-8000-000000000964'::uuid),
    ('00000000-0000-4000-8000-000000000965'::uuid, '00000000-0000-4000-8000-000000000966'::uuid),
    ('00000000-0000-4000-8000-000000000971'::uuid, '00000000-0000-4000-8000-000000000979'::uuid),
    ('00000000-0000-4000-8000-000000000973'::uuid, '00000000-0000-4000-8000-000000000974'::uuid),
    ('00000000-0000-4000-8000-000000000974'::uuid, '00000000-0000-4000-8000-000000000975'::uuid),
    ('00000000-0000-4000-8000-000000000975'::uuid, '00000000-0000-4000-8000-000000000976'::uuid),
    ('00000000-0000-4000-8000-000000000976'::uuid, '00000000-0000-4000-8000-000000000977'::uuid),
    ('00000000-0000-4000-8000-000000000977'::uuid, '00000000-0000-4000-8000-000000000978'::uuid),
    ('00000000-0000-4000-8000-000000000978'::uuid, '00000000-0000-4000-8000-000000000984'::uuid),
    ('00000000-0000-4000-8000-000000000979'::uuid, '00000000-0000-4000-8000-000000000985'::uuid),
    ('00000000-0000-4000-8000-000000000980'::uuid, '00000000-0000-4000-8000-000000000986'::uuid),
    ('00000000-0000-4000-8000-000000000983'::uuid, '00000000-0000-4000-8000-000000000987'::uuid)
) AS p (id, next)
WHERE e.id = p.id AND e.stable_id IS NULL;
