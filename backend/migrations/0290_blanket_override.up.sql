-- Manual blanket choice (Decke abweichend von der Regel): owners and admins can pick a blanket
-- (or "no blanket") for a horse that replaces the rule recommendation from day_from on, for
-- a number of nights (day_until inclusive) or until it is reset (day_until NULL). One choice
-- per horse; a new one replaces the old one. blanket_id NULL means "no blanket"; deleting the
-- chosen blanket drops the choice, so it never silently turns into "no blanket".
CREATE TABLE blanket_overrides (
    horse_id   uuid PRIMARY KEY REFERENCES horses (id) ON DELETE CASCADE,
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    blanket_id uuid REFERENCES blankets (id) ON DELETE CASCADE,
    day_from   date NOT NULL,
    day_until  date CHECK (day_until IS NULL OR day_until >= day_from),
    note       text,
    set_by     uuid REFERENCES users (id) ON DELETE SET NULL,
    set_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX blanket_overrides_stable_idx ON blanket_overrides (stable_id);
CREATE INDEX blanket_overrides_set_by_idx ON blanket_overrides (set_by);
