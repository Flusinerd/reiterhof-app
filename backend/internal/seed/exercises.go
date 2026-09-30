package seed

// Exercise library: global rows (stable_id NULL) shared by all stables. The content follows
// published sources (FN member magazine "Pferd und Mensch", FN Merkblätter for the badges
// Bodenarbeit, Vormustern and Longieren, de.wikipedia and established riding magazines); every
// entry lists the pages it is based on. Nothing here is made up: if a source gives no figure,
// the step leaves it out.
//
// Levels map the German class system: beginner = basic training and Klasse E/A, intermediate =
// A* to L*, advanced = L** and M (groundwork and lunging: the FN badge stages instead).
//
// IDs are seedID("8", n): 01-17 dressage, 20-29 jumping, 30-39 groundwork, 40-49 lunging.

type exerciseSeed struct {
	n          int
	discipline string
	level      string
	title      string
	tags       []string
	steps      []string
	next       int      // n of the follow-up exercise, 0 = none
	sources    []string // pages the exercise is based on (not stored)
}

const (
	fnMag = "https://magazin.pferdesport-deutschland.de/project/"
	// FN Merkblatt Bodenarbeit (Stand Oktober 2019)
	fnBodenarbeit = "https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf"
	// FN Merkblatt Abzeichen Longieren (Stand Oktober 2025)
	fnLongieren       = "https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf"
	longierenSinnvoll = "https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/"
)

// Library, ordered so that every follow-up exercise is inserted before its predecessor
// (next_exercise_id is a foreign key).
var exerciseLibrary = []exerciseSeed{
	// --- dressage ---------------------------------------------------------------------
	{14, "dressage", "intermediate", "Mitteltrab", []string{"schwung", "takt", "durchlaessigkeit"}, []string{
		"Aus einem gelösten, fleißigen Arbeitstrab zulegen; die Hände geben so weit vor, dass das Pferd seinen Rahmen verlängern kann.",
		"Der Raumgriff wird größer, ohne dass die Tritte eiliger werden; die Hinterhufe treten über die Spur der Vorderhufe.",
		"Der Hals bleibt der höchste Punkt, die Stirn-Nasen-Linie deutlich vor der Senkrechten.",
		"Mit halben Paraden zum Arbeitstrab zurückführen, das fleißige Abfußen bleibt erhalten.",
	}, 0, []string{fnMag + "lektion-im-fokus-07-2022/"}},
	{1, "dressage", "beginner", "Zügel-aus-der-Hand-kauen-lassen", []string{"losgelassenheit", "anlehnung", "takt"}, []string{
		"Im Arbeitstrab auf der Zirkellinie reiten, wenn das Pferd gelöst an den Hilfen steht.",
		"Die Hände allmählich vorgehen lassen, die Schenkel treiben weiter: Das Pferd dehnt sich vorwärts-abwärts, das Maul mindestens bis auf Höhe des Buggelenks.",
		"Die Verbindung zum Pferdemaul bleibt erhalten, Takt und Tempo bleiben gleich.",
		"Nach einer kurzen Reprise die Zügel wieder aufnehmen, damit das Pferd nicht auf die Vorhand kommt.",
	}, 14, []string{fnMag + "lektion-im-fokus-09-2021/", "https://www.wehorse.com/de/blog/zuegel-aus-der-hand-kauen-lassen/"}},
	{8, "dressage", "beginner", "Übergänge", []string{"durchlaessigkeit", "losgelassenheit"}, []string{
		"Schritt-Trab-Schritt an markierten Punkten reiten.",
		"Vor jedem Übergang eine halbe Parade geben; beim Durchparieren erst ausatmen, dann Gewicht, dann Schenkel und zuletzt Zügel.",
		"Nach dem Übergang den Takt sofort wiederfinden.",
		"Steigern: Trab-Galopp-Trab und Übergänge innerhalb der Gangart.",
	}, 14, []string{
		"https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-uebergaenge.html",
		fnMag + "10-tipps-11-12-2019/", fnMag + "lektion-im-fokus-07-2022/",
	}},
	{15, "dressage", "intermediate", "Einfacher Galoppwechsel", []string{"durchlaessigkeit", "versammlung", "balance"}, []string{
		"Die Übergänge Galopp-Schritt und Schritt-Galopp zunächst einzeln üben, anfangs mit einer längeren Schrittphase.",
		"Mit halben Paraden aus aktivem Galopp geschmeidig zum Mittelschritt durchparieren und drei bis fünf klare Schritte reiten.",
		"Dann unmittelbar und bergauf im neuen Galopp angaloppieren.",
		"Geeignete Linien: Zirkel verkleinern, halbe Volte, später durch die Länge der Bahn wechseln mit Wechsel über X.",
	}, 0, []string{fnMag + "lektion-im-fokus-02-2022/", fnMag + "10-tipps-05-2023/"}},
	{9, "dressage", "beginner", "Zirkel verkleinern und vergrößern", []string{"biegung", "losgelassenheit"}, []string{
		"Auf dem Zirkel im Arbeitstrab anreiten, Takt und Tempo halten.",
		"Spiralförmig verkleinern, pro Runde etwa eine Pferdebreite: Gewicht innen, der äußere Zügel und der verwahrende äußere Schenkel führen das Pferd nach innen.",
		"Je kleiner der Kreis, desto mehr das Tempo verkürzen; den nächstkleineren Kreis erst reiten, wenn der vorige gelingt.",
		"Mit dem vorwärts-seitwärts treibenden inneren Schenkel an den äußeren Zügel wieder vergrößern.",
	}, 15, []string{
		fnMag + "reiten-von-hufschlagfiguren-11-12-2023/", fnMag + "hufschlagfiguren-auf-der-richtigen-linie-02-2023",
		"https://www.tipps-zum-pferd.de/hufschlagfigur-zirkel-verkleinern-vergroessern_tipp_328.html",
	}},
	{16, "dressage", "intermediate", "Außengalopp", []string{"versammlung", "geraderichten", "balance"}, []string{
		"Voraussetzung ist ein ausbalancierter, geradegerichteter Handgalopp mit sicheren Übergängen.",
		"Zum Einstieg im Galopp eine einfache Schlangenlinie an der langen Seite reiten; das Pferd bleibt dabei im bisherigen Galopp.",
		"Innen bleibt die Seite, zu der das Pferd gestellt und gebogen ist; der innere Schenkel treibt an den äußeren Zügel.",
		"Später auf größeren Linien üben; zu viel Stellung und ein blockierender innerer Zügel stören.",
	}, 0, []string{fnMag + "10-tipps-08-2023/"}},
	{2, "dressage", "beginner", "Schlangenlinien durch die ganze Bahn", []string{"biegung", "losgelassenheit", "geraderichten"}, []string{
		"An der kurzen Seite beginnen; im 20 × 40-m-Viereck meist drei oder vier Bögen, dazwischen geradeaus parallel zur kurzen Seite.",
		"In jedem Bogen stellt der innere Zügel, der äußere gibt etwas nach und begrenzt; der innere Schenkel am Gurt biegt, der äußere hält die Hinterhand in der Spur.",
		"Beim Überqueren der Mittellinie das Pferd geraderichten und umstellen wie bei einem Handwechsel.",
		"Auf gleichmäßigen Takt und gleich große Bögen achten, nicht am inneren Zügel ziehen.",
	}, 16, []string{
		"https://www.pferderevue.at/content/pferderevue/pferderevue/de/magazin/ausbildung/2016/09/gymnastizierung_purmitschlangenlinieund-tour.html",
		"https://de.wikipedia.org/wiki/Hufschlagfigur", fnMag + "hufschlagfiguren-auf-der-richtigen-linie-02-2023",
	}},
	{13, "dressage", "intermediate", "Viereck verkleinern und vergrößern", []string{"durchlaessigkeit", "takt", "losgelassenheit"}, []string{
		"Am ersten Wechselpunkt der langen Seite im Mittelschritt oder Arbeitstrab beginnen.",
		"Mit Stellung, ohne Biegung vorwärts-seitwärts parallel zum Hufschlag bis zur Viertellinie; der innere Schenkel knapp hinter dem Gurt, der äußere verwahrt, der äußere Zügel verhindert das Ausfallen über die Schulter.",
		"Eine Pferdelänge geradeaus, dann im Schenkelweichen zurück zum zweiten Wechselpunkt.",
		"Der Takt steht im Vordergrund; zu viel Halsstellung und zu steiles Seitwärts vermeiden.",
	}, 0, []string{fnMag + "lektion-im-fokus-10-2021/", fnMag + "hufschlagfiguren-auf-der-richtigen-linie-02-2023"}},
	{4, "dressage", "beginner", "Schenkelweichen", []string{"losgelassenheit", "durchlaessigkeit", "takt"}, []string{
		"Im Mittelschritt oder Arbeitstrab entlang der Bande oder auf einer Diagonalen, mit Stellung, aber ohne Längsbiegung.",
		"Der innere Schenkel treibt knapp hinter dem Gurt vorwärts-seitwärts, der äußere verwahrt; der innere Zügel stellt, der äußere begrenzt die Halsstellung.",
		"Die inneren Beine treten vor und über die äußeren; die Abstellung bleibt unter 45 Grad, das Vorwärts bleibt erhalten.",
		"Fließend zwischen Schenkelweichen und Geradeausreiten wechseln.",
	}, 13, []string{fnMag + "10-tipps-08-2022/", "https://de.wikipedia.org/wiki/Schenkelweichen"}},
	{3, "dressage", "beginner", "Vorhandwendung", []string{"durchlaessigkeit", "losgelassenheit"}, []string{
		"Auf dem zweiten Hufschlag geschlossen halten, damit Kopf und Hals zur Bande Platz haben.",
		"Das Pferd in Richtung der Wendung stellen, nicht biegen; der neue innere Schenkel liegt seitwärtstreibend hinter dem Gurt.",
		"Mit kleinen Impulsen tritt die Hinterhand Schritt für Schritt im Kreisbogen um die Vorhand; äußerer Schenkel und äußerer Zügel verwahren.",
		"Bei Unsicherheit zwischendurch kurz halten, zum Abschluss wieder geschlossen halten.",
	}, 4, []string{"https://de.wikipedia.org/wiki/Vorhandwendung", fnMag + "diagonale-hilfengebung-verstehen-09-2023/"}},
	{7, "dressage", "intermediate", "Kurzkehrt", []string{"versammlung", "durchlaessigkeit", "biegung"}, []string{
		"Aus dem Mittelschritt mit einigen halben Paraden aufnehmen, aus dem Trab ein bis zwei Schritte vorher zum Schritt durchparieren, ohne zu halten.",
		"Das Pferd ist in Bewegungsrichtung gestellt und gebogen, die Vorhand beschreibt einen Halbkreis um die Hinterhand; das innere Hinterbein tritt fast auf der Stelle, der Viertakt bleibt klar.",
		"Vermehrt innen sitzen: Der innere Zügel leitet ein, der innere Schenkel erhält Fleiß und Biegung, der äußere Schenkel begrenzt, ohne herumzudrücken.",
		"Anfangs nur um 45 oder 90 Grad wenden, danach fleißig geradeaus reiten.",
	}, 0, []string{fnMag + "lektion-im-fokus-01-2022/"}},
	{6, "dressage", "beginner", "Rückwärtsrichten", []string{"durchlaessigkeit", "balance"}, []string{
		"Aus dem geschlossenen Halten auf gerader Linie beginnen.",
		"Beidseitig belastende Gewichtshilfe und vorwärtstreibende Schenkel; die annehmende oder aushaltende Zügelhilfe lenkt die Bewegung nach rückwärts.",
		"Das Pferd tritt im diagonalen Beinpaar rückwärts, in der Grundausbildung etwa eine Pferdelänge (drei bis vier Tritte).",
		"Zu starke Zügeleinwirkung vermeiden, sonst schleppt das Pferd die Füße.",
	}, 0, []string{fnMag + "lektion-im-fokus-11-12-2021/"}},
	{12, "dressage", "advanced", "Traversale", []string{"seitengaenge", "biegung", "versammlung", "durchlaessigkeit"}, []string{
		"Nur im versammelten Trab oder Galopp reiten; mit flachen, langen Traversalen beginnen, etwa einer halben Traversale von der Mittellinie zum Hufschlag.",
		"Das Pferd ist in Bewegungsrichtung gestellt und gebogen und geht fast parallel zur langen Seite, die Vorhand führt leicht.",
		"Gewicht innen, der innere Schenkel am Gurt, der äußere hinter dem Gurt treibt vorwärts-seitwärts; der innere Zügel stellt, der äußere begrenzt.",
		"Nicht mit dem äußeren Schenkel hinüberschieben, die Hinterhand darf nicht vorausgehen; zum Beenden geradeaus reiten.",
	}, 0, []string{fnMag + "lektion-im-fokus-10-2022/", "https://de.wikipedia.org/wiki/Traversale", "https://www.ehorses.de/magazin/traversale-reiten/"}},
	{17, "dressage", "advanced", "Renvers", []string{"seitengaenge", "biegung", "versammlung"}, []string{
		"Über eine einfache Schlangenlinie einsteigen: Sobald die Biegung entsteht, die Renvers-Hilfen geben.",
		"Die Hinterhand bleibt auf dem Hufschlag, die Vorhand wird in die Bahn geführt; das Pferd ist in Bewegungsrichtung gestellt und gebogen und geht auf vier Hufschlaglinien.",
		"Gewicht in Bewegungsrichtung, der innere Schenkel für die Biegung, der äußere führt die Hinterhand; der innere Zügel stellt, der äußere begrenzt.",
		"Zum Beenden die Vorhand zurück auf den Hufschlag führen; die Biegung nicht durch Halsstellung ersetzen.",
	}, 12, []string{fnMag + "lektion-im-fokus-05-2022/", "https://de.wikipedia.org/wiki/Traversale"}},
	{11, "dressage", "advanced", "Travers", []string{"seitengaenge", "biegung", "versammlung"}, []string{
		"An der langen Seite oder, bei jungen Pferden, auf dem Zirkel mit Stellung und Biegung in Bewegungsrichtung beginnen.",
		"Die Vorhand bleibt auf dem Hufschlag, die Hinterhand kommt etwa 30 bis 35 Grad herein; das Pferd geht auf vier Hufschlaglinien.",
		"Gewicht in Bewegungsrichtung, der innere Schenkel für die Biegung, der äußere bringt die Hinterhand herein; die innere Hand nicht über den Mähnenkamm drücken.",
		"Zum Beenden die Hinterhand zurück auf den Hufschlag führen und auf beiden Händen im Wechsel üben.",
	}, 17, []string{fnMag + "lektion-im-fokus-05-2022/", "https://de.wikipedia.org/wiki/Travers_(Reitsport)"}},
	{10, "dressage", "advanced", "Schulterherein", []string{"seitengaenge", "biegung", "versammlung", "geraderichten"}, []string{
		"Aus der Ecke heraus oder aus einer Volte beginnen und die Längsbiegung aus der Wendung mitnehmen.",
		"Die Vorhand etwa 30 Grad in die Bahn führen, die Hinterhand bleibt auf dem Hufschlag; das Pferd geht auf drei Hufschlaglinien, gleichmäßig um den inneren Schenkel gebogen.",
		"Gewicht innen, der innere Schenkel treibt und erhält die Biegung, der äußere begrenzt die Hinterhand; der äußere Zügel lässt die Stellung zu.",
		"Takt und Schwung haben Vorrang: nur eine halbe lange Seite reiten, dann geradeaus und loben.",
	}, 11, []string{fnMag + "lektion-im-fokus-03-2022/", "https://de.wikipedia.org/wiki/Schulterherein"}},
	{5, "dressage", "beginner", "Volte", []string{"biegung", "balance", "durchlaessigkeit"}, []string{
		"Vor der Volte mit einigen halben Paraden Takt, Gleichgewicht und Selbsthaltung sichern.",
		"Gewicht innen, der innere Schenkel für Längsbiegung und fleißiges Abfußen, der äußere verhindert das Ausweichen der Hinterhand; der äußere Zügel hält die Linie.",
		"Beim Abwenden zum Scheitelpunkt der Volte schauen, von dort zurück zum Ausgangspunkt.",
		"Zunächst größere Volten (10 m) reiten; ab Klasse L wird die 8-m-Volte verlangt.",
	}, 10, []string{fnMag + "lektion-im-fokus-07-2023/", fnMag + "lektion-im-fokus-03-2022/"}},

	// --- jumping: pole work, cavaletti, gymnastic jumping (distances for horses) ------
	{25, "jumping", "intermediate", "Distanzen reiten", []string{"rhythmus", "aufmerksamkeit", "durchlaessigkeit"}, []string{
		"Zwei Cavaletti auf gerader Linie im Abstand von 22 m aufstellen.",
		"Im Arbeitsgalopp mit sechs Galoppsprüngen dazwischen reiten und die Sprünge laut mitzählen.",
		"Klappen sechs Sprünge gleichmäßig, den Galopp auf sieben Sprünge verkürzen, danach auf fünf verlängern.",
		"Den Galopp schon vor dem ersten Cavaletti anpassen, danach nur den Rhythmus halten.",
	}, 0, []string{fnMag + "einstieg-in-das-springen-06-2021/", fnMag + "spruenge-passend-anreiten-05-2022"}},
	{24, "jumping", "intermediate", "Galoppcavaletti", []string{"rhythmus", "balance", "takt"}, []string{
		"Zwei oder mehr Cavaletti im Abstand von ca. 3 m aufstellen.",
		"Im ruhigen Arbeitsgalopp im leichten Sitz mittig anreiten.",
		"Im Fächer auf der Zirkellinie (innen ca. 2 m, außen ca. 3 m) die Galoppsprünge variieren: innen verkürzen, außen verlängern.",
		"Die höchste Cavaletti-Stufe nur im Galopp verwenden.",
	}, 25, []string{
		fnMag + "10-tipps-03-2023/", fnMag + "ausbildung-cavaletti-08-2024/",
		"https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-cavaletti.html",
	}},
	{23, "jumping", "intermediate", "Cavaletti auf dem Zirkel", []string{"rhythmus", "balance", "losgelassenheit"}, []string{
		"An beiden geschlossenen Zirkelseiten je zwei niedrige Cavaletti fächerförmig auslegen, in der Mitte 1,50 m Abstand; den ersten Hufschlag frei lassen.",
		"Leichttrabend auf dem 20-m-Zirkel über die Cavaletti traben, dann durch den Zirkel wechseln.",
		"Später vier Cavaletti im Fächer: Pferde mit weniger Raumgriff traben weiter innen, Pferde mit mehr Raumgriff weiter außen.",
	}, 24, []string{
		fnMag + "ausbildung-cavaletti-08-2024/",
		"https://www.equitana.com/essen/de-de/blog/dressur-springen-vielseitigkeit/lektionen-leicht-gemacht-mit-ingrid-klinke-cavaletti.html",
	}},
	{21, "jumping", "beginner", "Trabstangen", []string{"takt", "rhythmus", "losgelassenheit"}, []string{
		"Vier Stangen oder Cavaletti im Abstand von ca. 1,30 m (je nach Pferd 1,20–1,40 m) an der langen Seite auslegen.",
		"Mittig anreiten und leichttraben oder im leichten Sitz mitgehen.",
		"Das Pferd gleichmäßig, schwungvoll und im Rhythmus weitertraben lassen.",
		"Später auf sechs Stangen erweitern und auf gebogener Linie reiten.",
	}, 23, []string{fnMag + "10-tipps-03-2023/", fnMag + "ausbildung-cavaletti-08-2024/", "https://reiter-pferde.de/das-geht-locker-loesungsarbeit-mit-stangen-und-cavaletti/"}},
	{20, "jumping", "beginner", "Schrittstangen", []string{"takt", "losgelassenheit"}, []string{
		"Sechs bis acht Stangen im Abstand von ca. 0,80 m (Großpferd 0,80–0,90 m) auf gerader Linie auslegen und mit dem Maßband nachmessen.",
		"Im gleichmäßigen Schritt mittig über die Stangen reiten.",
		"Die Stangen regeln Takt und Gleichmaß, besonders bei eiligem Schritt.",
	}, 21, []string{fnMag + "10-tipps-03-2023/", fnMag + "ausbildung-cavaletti-08-2024/", "https://de.wikipedia.org/wiki/Cavaletti_(Reitsport)"}},
	{27, "jumping", "advanced", "Oxerreihe", []string{"springgymnastik", "rhythmus", "aufmerksamkeit"}, []string{
		"Vier Hoch-Weit-Sprünge in einer Reihe aufbauen: 6,50–7,00 m für einen Galoppsprung, 9,70–10,20 m für zwei.",
		"Aus ruhigem Tempo mittig anreiten und gerade springen; das Pferd soll energisch abspringen, nicht aus höherem Tempo.",
		"Springreihen nur dosiert einsetzen, Einsprung-Aussprung-Reihen kosten viel Kraft.",
	}, 0, []string{fnMag + "springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017"}},
	{22, "jumping", "intermediate", "Gymnastikreihe", []string{"springgymnastik", "rhythmus", "balance"}, []string{
		"Aufbau: Cavaletti, 2,20 m bis zum Kreuz, 3,40 m bis zum zweiten Kreuz (Einsprung-Aussprung), 6,80 m bis zum Steilsprung, 10,00 m bis zum Oxer.",
		"Zunächst nur die ersten Elemente sehr niedrig aufbauen und die Reihe schrittweise erweitern.",
		"Aus dem Trab anreiten, im leichten Sitz mitgehen und mit der Hand der Kopf-Hals-Bewegung folgen.",
		"Die Abstände sind Zirka-Werte für Großpferde; je nach Pferdegröße und Hindernishöhe anpassen, für Ponys verkürzen.",
	}, 27, []string{
		fnMag + "springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017",
		fnMag + "einstieg-in-das-springen-06-2021/", fnMag + "springausbildung-fuer-reiter-und-pferd-teil-4-07-2017",
	}},
	{26, "jumping", "beginner", "Kreuzsprung mit Vorlegestange", []string{"springgymnastik", "balance"}, []string{
		"Ein Kreuz aufbauen und die Vorlegestange ca. 2,20–2,50 m davor parallel auslegen; bei unerfahrenen Pferden Fangständer verwenden.",
		"Im gleichmäßigen Trab mittig anreiten und im leichten Sitz mitgehen.",
		"Die Linie vor und nach dem Sprung einhalten; Pylonen zeigen den Weg.",
		"Aus dem Galopp liegt die Vorlegestange ca. 3 m vor dem Sprung.",
	}, 22, []string{
		fnMag + "springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017",
		fnMag + "einstieg-in-das-springen-06-2021/",
	}},

	// --- groundwork (FN badges Bodenarbeit, Vormustern) --------------------------------
	{36, "groundwork", "advanced", "Wendung um die Hinterhand an der Hand", []string{"durchlaessigkeit", "balance", "biegung"}, []string{
		"Der Pferdekopf bleibt beim Führenden, die Vorhand wird um die Hinterhand herumgeführt.",
		"Das äußere Vorderbein kreuzt über das innere.",
		"Die Hinterbeine treten weiter mit und bleiben nicht stehen.",
		"Mit 90 Grad beginnen und auf 180 oder 360 Grad steigern.",
	}, 0, []string{fnBodenarbeit}},
	{35, "groundwork", "intermediate", "Seitwärts weichen an der Hand", []string{"durchlaessigkeit", "balance"}, []string{
		"An der Bande oder auf einer vorgegebenen Linie beginnen.",
		"In Schulterhöhe stehen und erst die Schulter, dann die Hinterhand weichen lassen.",
		"Auf gleichmäßiges Übertreten achten.",
		"Das Pferd weicht willig und flüssig; Körpersprache, Berührung und Stimme dosiert einsetzen.",
	}, 36, []string{fnBodenarbeit, "https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/"}},
	{34, "groundwork", "beginner", "Vorhand weichen lassen", []string{"durchlaessigkeit", "aufmerksamkeit"}, []string{
		"Den Pferdekopf halten und leicht entgegen der Bewegungsrichtung stellen.",
		"Mit auffordernder Körpersprache und Stimme um die Vorhand herumtreten lassen, bei Bedarf hinter der Gurtlage berühren.",
		"Das innere Hinterbein kreuzt vor dem äußeren.",
		"Sofort nachgeben, sobald das Pferd weicht.",
	}, 35, []string{fnBodenarbeit, "https://www.loesdau.de/specials/loesdau-lessons/bodenarbeit-mit-sabine-ellinger/teil-3-die-zweite-grunduebung-der-bodenarbeit/"}},
	{30, "groundwork", "beginner", "Führen und Halten", []string{"vertrauen", "aufmerksamkeit"}, []string{
		"Links vom Pferd zwischen Pferdekopf und Schulter gehen, den Strick in der rechten Hand bis zu 50 cm unterhalb des Halfters.",
		"Gleichzeitig antreten; das Pferd richtet sich nach dem Tempo des Führenden.",
		"Das Anhalten über die Körpersprache einleiten, bei Bedarf mit Stimme oder leichten Impulsen am Strick.",
		"Punktgenau halten, möglichst ohne Handeinwirkung und mit durchhängendem Strick.",
	}, 34, []string{fnBodenarbeit}},
	{37, "groundwork", "intermediate", "Vorführen auf der Dreiecksbahn", []string{"takt", "aufmerksamkeit"}, []string{
		"Das Pferd 3–4 m vor dem Betrachter offen aufstellen, sodass alle vier Beine zu sehen sind.",
		"Auf Trense führen: beide Zügel 3–4 Handbreit hinter den Gebissringen in der rechten Hand, leicht durchhängend; der Führende geht links.",
		"Die erste Seite im Schritt führen, nach der Wendung antraben und vor der nächsten Wendung zum Schritt durchparieren; immer nach rechts wenden.",
		"Zum Schluss vorbeiführen, rechts wenden und von der anderen Seite offen aufstellen.",
	}, 0, []string{"https://www.renate-gassner.de/uploads/media/Merkblatt_Vormustern_.pdf", fnBodenarbeit}},
	{31, "groundwork", "intermediate", "Rückwärtsrichten an der Hand", []string{"aufmerksamkeit", "balance"}, []string{
		"Aus dem Halten mit der Stimme ankündigen („Zurück“).",
		"Mit einem Impuls am Halfter auffordern, bei Bedarf mit leichtem Druck der Hand am Buggelenk unterstützen.",
		"Sofort nachgeben, sobald das Pferd zurücktritt; schon die Tendenz nach hinten belohnen.",
		"Eine Pferdelänge gerade und im Zweitakt rückwärts, dann wieder halten.",
	}, 37, []string{fnBodenarbeit, fnMag + "10-tipps-05-2020", "https://www.cavallo.de/reittraining/das-bringt-rueckwaertsrichten-dem-pferd/"}},
	{33, "groundwork", "beginner", "Anbinden und Stillstehen", []string{"vertrauen", "aufmerksamkeit"}, []string{
		"Den Anbindestrick mit Panikhaken am Halfter einhaken.",
		"Mit Sicherheitsknoten anbinden, ca. 60–80 cm zwischen Panikhaken und Anbindering.",
		"So viel Spielraum lassen, dass das Pferd sich bewegen, aber nicht in den Strick treten kann.",
		"Das Pferd steht ruhig auf allen vier Beinen, bis ein neues Signal kommt.",
	}, 0, []string{fnBodenarbeit}},
	{38, "groundwork", "intermediate", "Stangenarbeit an der Hand", []string{"aufmerksamkeit", "balance", "vertrauen"}, []string{
		"Etwa 3 m vor der Stange das Tempo verlangsamen.",
		"So anhalten, dass die Vorhand vor und die Hinterhand hinter der Stange steht.",
		"Im Stangenlabyrinth im Schritt führen, ohne Stangen zu berühren oder seitlich auszubrechen.",
		"Bei Schwierigkeiten die Aufgabe leichter machen und mit einer gelungenen Ausführung enden.",
	}, 0, []string{fnBodenarbeit, "https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/"}},
	{39, "groundwork", "intermediate", "Über die Plane führen", []string{"vertrauen", "gelassenheit"}, []string{
		"Zuerst in der Halle oder auf einem umzäunten Platz üben.",
		"Der Führende geht zwischen Plane und Pferd und hält dem Pferd den Fluchtweg vorwärts-seitwärts frei.",
		"Den Reiz nur so weit steigern, dass das Pferd nicht flüchtet; Neugier und Untersuchen zulassen.",
		"Selbst ruhig bleiben und jeden Schritt zur Plane mit einer Pause belohnen.",
	}, 0, []string{fnBodenarbeit, "https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/"}},
	{32, "groundwork", "advanced", "Freiarbeit", []string{"vertrauen", "aufmerksamkeit"}, []string{
		"In einer umzäunten Umgebung wie Reitplatz oder Halle arbeiten.",
		"Am Strick beginnen und die Hilfen über Stimme und Körpersprache aufbauen, Strick und Gerte unterstützen.",
		"Reagiert das Pferd sicher, den Strick abnehmen und die Hilfsmittel nach und nach weglassen.",
	}, 0, []string{"https://www.cavallo.de/reittraining/freiarbeit-vertieft-die-freundschaft-zum-pferd/"}},

	// --- lunging (FN badge Longieren, FN magazine "Longieren – aber sinnvoll") ---------
	{43, "lunge", "intermediate", "Zirkel verkleinern, vergrößern und verlagern", []string{"biegung", "balance", "durchlaessigkeit"}, []string{
		"Im Arbeitstrab den Zirkel allmählich verkleinern und wieder vergrößern.",
		"Takt, Losgelassenheit und Anlehnung dabei erhalten.",
		"Den Zirkel verlagern und dabei die Trabtritte zweimal verlängern und verkürzen.",
		"Im Galopp die Galoppsprünge zweimal verlängern und verkürzen.",
	}, 0, []string{fnLongieren}},
	{41, "lunge", "beginner", "Übergänge an der Longe", []string{"durchlaessigkeit", "aufmerksamkeit", "takt"}, []string{
		"Übergänge Schritt–Trab–Galopp und zurück bis zum Halten longieren.",
		"Eine tiefe, ruhige Stimme zum Zurückholen, eine höhere für mehr Aufmerksamkeit; es zählt der Tonfall, nicht die Lautstärke.",
		"Zum Durchparieren Paraden an der Longe mit einem klaren Kommando verbinden.",
		"Eine neue Gangart erst verlangen, wenn das Pferd in der aktuellen ausbalanciert ist.",
	}, 43, []string{fnLongieren, longierenSinnvoll, fnMag + "longieren-aber-sinnvoll-08-2019/"}},
	{40, "lunge", "beginner", "Aufwärmen im Schritt", []string{"losgelassenheit", "takt", "vertrauen"}, []string{
		"Mit Handschuhen und festem Schuhwerk, ohne Sporen longieren; die Longe hat Handschlaufe und Karabiner, keinen Wirbelkarabiner.",
		"Auf einem umzäunten Zirkel mit mindestens 16 m Durchmesser und rutschfestem Boden arbeiten.",
		"Longe, Peitsche und Pferd bilden ein Dreieck; das Pferd Runde für Runde auf die große Zirkellinie hinausgehen lassen.",
		"10–15 Minuten unausgebunden im Schritt aufwärmen; die ganze Einheit dauert höchstens 35–40 Minuten.",
	}, 41, []string{fnMag + "longieren-aber-sinnvoll-08-2019/", longierenSinnvoll, fnLongieren}},
	{42, "lunge", "beginner", "Handwechsel an der Longe", []string{"aufmerksamkeit", "durchlaessigkeit"}, []string{
		"Das Pferd mit der Stimme aufmerksam machen und in ruhiger Haltung nach innen holen, die Longe bleibt leicht gespannt.",
		"Longe und Peitsche die Hand wechseln lassen und die Ausbindezügel für die neue Hand anpassen.",
		"Auf der neuen Hand wieder hinauslongieren.",
		"Häufig die Hand wechseln, das ist an der Longe besonders wichtig.",
	}, 0, []string{fnMag + "longieren-aber-sinnvoll-08-2019/", longierenSinnvoll, "https://www.cavallo.de/reittraining/gymnastik-an-der-longe/"}},
	{44, "lunge", "intermediate", "Longieren über Trabstangen", []string{"takt", "balance", "losgelassenheit"}, []string{
		"Mit einer Stange beginnen; die nächste erst dazulegen, wenn das Pferd die erste gelassen überwindet.",
		"Die Abstände mit dem Maßband messen: Schritt ca. 0,80–0,90 m, Trab ca. 1,20–1,40 m.",
		"Vier Stangen sind ideal, bei fünf oder sechs ist Schluss.",
		"Stangenarbeit regelmäßig, aber höchstens alle zwei Tage einplanen.",
	}, 0, []string{fnMag + "ausbildung-cavaletti-08-2024/", longierenSinnvoll}},
	{45, "lunge", "intermediate", "Longieren mit Ausbindezügeln", []string{"anlehnung", "losgelassenheit", "takt"}, []string{
		"Die Ausbindezügel erst nach dem unausgebundenen Aufwärmen im Schritt einschnallen.",
		"Länge prüfen: vor dem Pferd stehen, das Gebiss auf beiden Seiten fassen und den Kopf vorsichtig heranführen, bis beide Zügel gleichmäßig leicht anstehen; die Stirn-Nasen-Linie bleibt vor der Senkrechten.",
		"Bei jedem Handwechsel anpassen; den Trensenzügel nie zum Ausbinden verwenden.",
		"Zum Schluss etwa 10 Minuten unausgebunden im Schritt entspannen lassen.",
	}, 0, []string{fnMag + "longieren-aber-sinnvoll-08-2019/", longierenSinnvoll, fnLongieren}},
}
