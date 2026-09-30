-- M6 Training: link a session to the library exercise that was practised (focus rating
-- "Sitzt" then suggests the next progression) and add integrity checks on session values.

ALTER TABLE sessions ADD COLUMN exercise_id uuid REFERENCES exercises (id) ON DELETE SET NULL;
CREATE INDEX sessions_exercise_idx ON sessions (horse_id, exercise_id) WHERE exercise_id IS NOT NULL;

ALTER TABLE sessions ADD CONSTRAINT sessions_activity_check
    CHECK (activity IN ('hall', 'arena', 'hack', 'lunge', 'jumping', 'groundwork', 'walker'));
ALTER TABLE sessions ADD CONSTRAINT sessions_feel_check
    CHECK (feel IS NULL OR feel IN ('fresh', 'loose', 'tired', 'tense'));
ALTER TABLE sessions ADD CONSTRAINT sessions_focus_rating_check
    CHECK (focus_rating IS NULL OR focus_rating BETWEEN 1 AND 3);
