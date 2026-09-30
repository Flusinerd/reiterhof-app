package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// insertHorseCare adds emergency cards, extra contacts, health items and a phone number
// for the demo horses (JAN-49, JAN-12). Due dates are relative to today so the demo always
// shows a mix of overdue, soon and later items. Like the rest of the seed it never
// overwrites values that are already set.
func insertHorseCare(ctx context.Context, tx pgx.Tx) error {
	var b pgx.Batch
	q := func(sql string, args ...any) { b.Queue(sql, args...) }

	// Palette keys from the design system instead of coat colors.
	colors := map[string]string{
		HorseLuna: "green", HorseFanta: "amber", HorseBalu: "blue", HorseCookie: "rose",
		HorseMerlin: "violet", HorsePepe: "teal", HorseNala: "amber",
	}
	for id, key := range colors {
		q(`UPDATE horses SET color_key = $2 WHERE id = $1 AND color_key IN ('brown','chestnut','black','palomino','grey','bay')`, id, key)
	}

	q(`UPDATE users SET phone = '+49 170 1234567' WHERE id = $1 AND phone IS NULL`, UserJan)
	q(`UPDATE users SET phone = '+49 171 7654321' WHERE id = $1 AND phone IS NULL`, UserAnna)

	q(`UPDATE horses SET breed = 'Hannoveraner', weight_kg = 580, vet_name = 'Tierarztpraxis Dr. Berger', vet_phone = '+49 2362 55501',
			emergency_note = 'Neigt zu Koliken. Bei Unruhe, Scharren oder Wälzen sofort Tierarzt anrufen und Halter informieren.',
			emergency_medication = 'Buscopan 20 ml i.v. (nur vom Tierarzt)', allergies = 'Keine bekannt',
			insurance = 'OP-Versicherung, Police 4711-22', helper_note = 'Frisst zuerst Heu, dann Kraftfutter. Auf der Weide nie mit Fanta zusammen.'
		WHERE id = $1 AND emergency_note IS NULL`, HorseLuna)
	q(`UPDATE horses SET breed = 'Haflinger', weight_kg = 430, vet_name = 'Tierarztpraxis Dr. Berger', vet_phone = '+49 2362 55501',
			emergency_note = 'Hufrehe-gefährdet: kein frisches Gras, Kraftfutter nur nach Absprache.',
			permanent_medication = 'Pergolid 1 mg täglich (Cushing)', allergies = 'Penicillin'
		WHERE id = $1 AND emergency_note IS NULL`, HorseFanta)
	q(`UPDATE horses SET breed = 'Deutsches Reitpony', weight_kg = 320, vet_name = 'Tierklinik Dorsten', vet_phone = '+49 2362 55599'
		WHERE id = $1 AND vet_name IS NULL`, HorseBalu)

	contacts := []struct{ id, horse, label, name, phone string }{
		{"00000000-0000-4000-8000-000000000901", HorseLuna, "Tierklinik (Notdienst)", "Tierklinik Dorsten", "+49 2362 55599"},
		{"00000000-0000-4000-8000-000000000902", HorseLuna, "Vertretung", "Mia", "+49 172 5550123"},
		{"00000000-0000-4000-8000-000000000903", HorseFanta, "Hufschmied", "Hufbeschlag Kruse", "+49 173 5550456"},
	}
	for _, c := range contacts {
		q(`INSERT INTO emergency_contacts (id, stable_id, horse_id, label, name, phone) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
			c.id, StableB, c.horse, c.label, c.name, c.phone)
	}

	// offset: days from today until due (negative = overdue).
	items := []struct {
		id, horse, kind, label string
		offset, interval       *int
		dailyTime              *string
		note                   string
	}{
		{"00000000-0000-4000-8000-000000000801", HorseLuna, "vaccination", "Influenza und Tetanus", n(120), n(365), nil, ""},
		{"00000000-0000-4000-8000-000000000802", HorseLuna, "farrier", "Hufschmied", n(6), n(42), nil, "Termin bei Kruse, Vormittag"},
		{"00000000-0000-4000-8000-000000000803", HorseLuna, "deworming", "Wurmkur", n(-9), n(90), nil, "Nach Kotprobe"},
		{"00000000-0000-4000-8000-000000000804", HorseLuna, "dentist", "Zahnkontrolle", n(200), n(365), nil, ""},
		{"00000000-0000-4000-8000-000000000805", HorseFanta, "vaccination", "Influenza", n(20), n(180), nil, ""},
		{"00000000-0000-4000-8000-000000000806", HorseFanta, "farrier", "Hufschmied", n(1), n(35), nil, ""},
		{"00000000-0000-4000-8000-000000000807", HorseFanta, "medication", "Pergolid 1 mg", nil, nil, s("08:00"), "Ins Futter mischen"},
		{"00000000-0000-4000-8000-000000000808", HorseFanta, "physio", "Physiotherapie", n(14), n(28), nil, ""},
		{"00000000-0000-4000-8000-000000000809", HorseBalu, "farrier", "Hufschmied", n(30), n(56), nil, ""},
	}
	for _, it := range items {
		var due any
		if it.offset != nil {
			due = *it.offset
		}
		q(`INSERT INTO health_items (id, stable_id, horse_id, kind, label, due_date, interval_days, note, daily_time)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6::int IS NULL THEN NULL ELSE current_date + $6::int END, $7, NULLIF($8, ''), $9::time)
			ON CONFLICT DO NOTHING`,
			it.id, StableB, it.horse, it.kind, it.label, due, it.interval, it.note, it.dailyTime)
	}

	res := tx.SendBatch(ctx, &b)
	for range b.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return fmt.Errorf("seed horse care: %w", err)
		}
	}
	return res.Close()
}

func n(v int) *int { return &v }
