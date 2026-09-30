package health

import (
	"context"
	"slices"
	"sort"
	"time"
)

// BundleKinds are the kinds where one appointment for several horses saves the vet a trip
// (JAN-54): general vet, vaccination and dentist.
var BundleKinds = []string{"vet", "vaccination", "dentist"}

// BundleWindowDays is both the look-ahead of an item ("approaching": due within this many
// days) and the tolerance around its due date in which other horses count as "also due".
const BundleWindowDays = 14

// Bundle tells that other horses of the stable have the same kind of appointment due at about
// the same time, so the appointment can be bundled.
type Bundle struct {
	Kind string `json:"kind"`
	// Count is the number of other horses (len(Horses)).
	Count  int           `json:"count"`
	Horses []BundleHorse `json:"horses"`
}

// BundleHorse is one other horse that is also due.
type BundleHorse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// candidate is a dated item of another horse.
type candidate struct {
	horseID, horseName, kind string
	due                      time.Time
}

// bundleEligible reports whether an item is a bundling candidate (see computeBundle).
func bundleEligible(it Item) bool {
	return it.DueDate != nil && it.DaysUntilDue != nil && it.DailyTime == nil && contains(BundleKinds, it.Kind) &&
		*it.DaysUntilDue >= 0 && *it.DaysUntilDue <= BundleWindowDays
}

// computeBundle returns the bundle of an item or nil. The item qualifies when its kind is in
// BundleKinds and it is due within the next BundleWindowDays days (today included, overdue
// items do not count as approaching). Other horses count when they have an item of the same
// kind due within +-BundleWindowDays days of this item's due date (overdue included).
func computeBundle(it Item, others []candidate) *Bundle {
	if !bundleEligible(it) {
		return nil
	}
	due, err := time.Parse("2006-01-02", *it.DueDate)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	b := &Bundle{Kind: it.Kind, Horses: []BundleHorse{}}
	for _, o := range others {
		if o.kind != it.Kind || o.horseID == it.HorseID || seen[o.horseID] {
			continue
		}
		if diff := int(o.due.Sub(due).Hours() / 24); diff < -BundleWindowDays || diff > BundleWindowDays {
			continue
		}
		seen[o.horseID] = true
		b.Horses = append(b.Horses, BundleHorse{ID: o.horseID, Name: o.horseName})
	}
	if len(b.Horses) == 0 {
		return nil
	}
	sort.Slice(b.Horses, func(i, j int) bool {
		if b.Horses[i].Name != b.Horses[j].Name {
			return b.Horses[i].Name < b.Horses[j].Name
		}
		return b.Horses[i].ID < b.Horses[j].ID
	})
	b.Count = len(b.Horses)
	return b
}

// annotateBundles sets Bundle on the items of one horse. One query loads the dated items of
// the same kinds of all other horses of the stable in the relevant date range.
func (h *handler) annotateBundles(ctx context.Context, stableID, horseID string, today time.Time, items []Item) error {
	if !slices.ContainsFunc(items, bundleEligible) {
		return nil
	}
	rows, err := h.deps.Pool.Query(ctx, `SELECT i.horse_id, o.name, i.kind, i.due_date
		FROM health_items i JOIN horses o ON o.id = i.horse_id AND o.stable_id = i.stable_id
		WHERE i.stable_id = $1 AND i.horse_id <> $2 AND i.kind = ANY($3) AND i.daily_time IS NULL
		  AND i.due_date BETWEEN $4::date - $6::int AND $4::date + $5::int + $6::int`,
		stableID, horseID, BundleKinds, today, BundleWindowDays, BundleWindowDays)
	if err != nil {
		return err
	}
	defer rows.Close()
	var others []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.horseID, &c.horseName, &c.kind, &c.due); err != nil {
			return err
		}
		others = append(others, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range items {
		items[i].Bundle = computeBundle(items[i], others)
	}
	return nil
}
