-- "Wie war's?" after the ride: how the session felt for the rider (hard, good, easy). Separate from
-- sessions.feel, which is how the horse was (fresh, loose, tired, tense, used by the recommender).
-- Nullable: sessions of older app versions and quick logs have none.
ALTER TABLE sessions ADD COLUMN feeling text;
ALTER TABLE sessions ADD CONSTRAINT sessions_feeling_check
    CHECK (feeling IS NULL OR feeling IN ('hard', 'good', 'easy'));
