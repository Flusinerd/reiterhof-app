-- Schenkelweichen (leg-yield) was hard to find: the dressage row 804 lacked the aim
-- "Seitengänge", so the library filter and the AI plan's tag matching skipped it, and horses
-- whose hall and arena days use the jumping library (jumping horses) had no leg-yield at all.
-- Tags of 804 and of 813 (Viereck verkleinern und vergrößern, which is ridden in leg-yield) gain
-- "seitengaenge"; the jumping library gets its own Schenkelweichen for the flatwork of jumpers,
-- based on the same sources as 804. Global rows only (stable_id NULL), idempotent.

UPDATE exercises SET goal_tags = goal_tags || '{seitengaenge}'::text[]
WHERE id IN ('00000000-0000-4000-8000-000000000804', '00000000-0000-4000-8000-000000000813')
  AND stable_id IS NULL AND NOT ('seitengaenge' = ANY (goal_tags));

-- FN_MAG/10-tipps-08-2022/ (https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/)
-- https://de.wikipedia.org/wiki/Schenkelweichen
INSERT INTO exercises (id, stable_id, discipline, level, goal_tags, title, steps) VALUES
('00000000-0000-4000-8000-000000000846', NULL, 'jumping', 'beginner', '{seitengaenge,losgelassenheit,durchlaessigkeit}', 'Schenkelweichen',
   '[
     "In der Lösungsphase vor der Springarbeit im Mittelschritt oder Arbeitstrab, entlang der Bande oder auf einer Diagonalen, mit Stellung, aber ohne Längsbiegung.",
     "Der innere Schenkel treibt knapp hinter dem Gurt vorwärts-seitwärts, der äußere verwahrt; der innere Zügel stellt, der äußere begrenzt die Halsstellung.",
     "Die inneren Beine treten vor und über die äußeren; die Abstellung bleibt unter 45 Grad, das Vorwärts bleibt erhalten.",
     "Fließend zwischen Schenkelweichen und Geradeausreiten wechseln."
   ]')
ON CONFLICT (id) DO UPDATE SET
    discipline = EXCLUDED.discipline, level = EXCLUDED.level, goal_tags = EXCLUDED.goal_tags,
    title = EXCLUDED.title, steps = EXCLUDED.steps
WHERE exercises.stable_id IS NULL;

-- In the jumping library it leads on to the trot poles, like the dressage row leads to 813.
UPDATE exercises SET next_exercise_id = '00000000-0000-4000-8000-000000000821'
WHERE id = '00000000-0000-4000-8000-000000000846' AND stable_id IS NULL;
