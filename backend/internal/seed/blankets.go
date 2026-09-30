package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// insertBlankets completes the blanket demo data (M3): a storage location for every
// blanket and blankets with rules for the horses that have none yet. Existing rows and
// values are never overwritten.
func insertBlankets(ctx context.Context, tx pgx.Tx) error {
	var b pgx.Batch
	q := func(sql string, args ...any) { b.Queue(sql, args...) }

	locations := []struct{ id, location string }{
		{BlanketLunaRain, "Haken 3"}, {BlanketLuna100, "Haken 4"}, {BlanketLuna200, "Sattelkammer, Regal 2"},
		{BlanketBalu150, "Haken 7"},
	}
	for _, l := range locations {
		q(`UPDATE blankets SET location = $2 WHERE id = $1 AND location IS NULL`, l.id, l.location)
	}

	// One blanket per remaining horse; rules: below 5 degrees the blanket, otherwise none.
	others := []struct {
		id, horse, name, color, location string
		fillG                            int
		rule1, rule2                     string
	}{
		{BlanketFanta100, HorseFanta, "Decke 100 g", "blau", "Haken 5", 100, "00000000-0000-4000-8000-000000000508", "00000000-0000-4000-8000-000000000509"},
		{BlanketCookie150, HorseCookie, "Decke 150 g", "rot", "Haken 6", 150, "00000000-0000-4000-8000-000000000510", "00000000-0000-4000-8000-000000000511"},
		{BlanketMerlin100, HorseMerlin, "Decke 100 g", "grün", "Haken 8", 100, "00000000-0000-4000-8000-000000000512", "00000000-0000-4000-8000-000000000513"},
		{BlanketPepe200, HorsePepe, "Decke 200 g", "schwarz", "Haken 9", 200, "00000000-0000-4000-8000-000000000514", "00000000-0000-4000-8000-000000000515"},
		{BlanketNala100, HorseNala, "Decke 100 g", "grau", "Haken 10", 100, "00000000-0000-4000-8000-000000000516", "00000000-0000-4000-8000-000000000517"},
	}
	for _, o := range others {
		q(`INSERT INTO blankets (id, stable_id, horse_id, name, fill_g, color, location) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`,
			o.id, StableB, o.horse, o.name, o.fillG, o.color, o.location)
		q(`INSERT INTO blanket_rules (id, stable_id, horse_id, position, temp_min, temp_max, rain, blanket_id, note) VALUES ($1, $2, $3, 1, NULL, 5, NULL, $4, NULL) ON CONFLICT DO NOTHING`,
			o.rule1, StableB, o.horse, o.id)
		q(`INSERT INTO blanket_rules (id, stable_id, horse_id, position, temp_min, temp_max, rain, blanket_id, note) VALUES ($1, $2, $3, 2, NULL, NULL, NULL, NULL, 'Keine Decke') ON CONFLICT DO NOTHING`,
			o.rule2, StableB, o.horse)
	}

	res := tx.SendBatch(ctx, &b)
	for range b.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return fmt.Errorf("seed blankets: %w", err)
		}
	}
	return res.Close()
}
