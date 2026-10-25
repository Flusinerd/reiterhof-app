package horses

import (
	"context"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// EventChanged is the realtime event after a horse, its riders or its emergency contacts changed:
// create, edit (name, photo, owner, ...), rider set or removed, contact added, edited or removed.
// Data: {horse_id}. The apps refetch the horse lists and everything derived from names, owners and
// riders (training horses, blanket board, roles).
const EventChanged = "horse.changed"

// hint sends EventChanged after the change was written. The data is committed by then, so a failure
// is only logged.
func (h *handler) hint(ctx context.Context, stableID, horseID string) {
	if err := realtime.Publish(ctx, h.deps.Pool, stableID, EventChanged, map[string]any{"horse_id": horseID}); err != nil {
		h.deps.Log.WarnContext(ctx, "horses: publish event", "err", err)
	}
}
