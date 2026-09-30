package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Observation IDs (table digit "c" of the scheme in ids.go). The Fanta observation is the one
// the reha plan of Fanta refers to.
const (
	ObservationFanta      = "00000000-0000-4000-8000-000000000c01"
	ObservationLunaCough  = "00000000-0000-4000-8000-000000000c02"
	ObservationBaluDone   = "00000000-0000-4000-8000-000000000c03"
	ObservationLunaOldEat = "00000000-0000-4000-8000-000000000c04"
)

// insertObservations adds a few reports (JAN-50, JAN-52): open and finished ones, dated
// relative to now. Like the rest of the seed it never overwrites existing rows.
func insertObservations(ctx context.Context, tx pgx.Tx, now time.Time) error {
	rows := []struct {
		id, horse, user, category, bodyPart, description, urgency, status string
		daysAgo                                                           int
	}{
		{ObservationFanta, HorseFanta, UserLea, "lameness", "front_right", "Nimmt das rechte Vorderbein beim Antraben nicht voll auf, Huf ist warm.", "check", "watch", 12},
		{ObservationLunaCough, HorseLuna, UserMia, "cough", "head", "Hustet beim Anreiten, danach besser.", "info", "watch", 1},
		{ObservationBaluDone, HorseBalu, UserKai, "blanket_equipment", "back", "Decke rutscht nach hinten, Gurt nachgestellt.", "info", "done", 8},
		{ObservationLunaOldEat, HorseLuna, UserTom, "not_eating", "", "Hat das Heu am Abend liegen lassen, am nächsten Morgen wieder normal.", "check", "done", 20},
	}
	var b pgx.Batch
	for _, r := range rows {
		b.Queue(`INSERT INTO observations (id, stable_id, horse_id, reported_by, category, body_part, description, urgency, status, created_at)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10) ON CONFLICT DO NOTHING`,
			r.id, StableB, r.horse, r.user, r.category, r.bodyPart, r.description, r.urgency, r.status, now.AddDate(0, 0, -r.daysAgo))
	}
	res := tx.SendBatch(ctx, &b)
	for range b.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return fmt.Errorf("seed observations: %w", err)
		}
	}
	return res.Close()
}
