package trainingapi

import (
	"context"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// EventChanged is the realtime event after the training data of a horse changed: a session was
// logged, a week day was taken, released or planned, the profile was saved. Data: {horse_id}.
// The apps refetch the week, today, diary and reminders of that horse (docs/architecture.md,
// "Realtime").
const EventChanged = "training.changed"

// publishChanged sends EventChanged on q; with a transaction the event goes out on commit.
func publishChanged(ctx context.Context, q realtime.Execer, a access) error {
	return realtime.Publish(ctx, q, a.user.StableID, EventChanged, map[string]any{"horse_id": a.horseID})
}

// hintChanged sends EventChanged after a write that has no transaction of its own. The data is
// committed by then, so a failure is only logged.
func (h *handler) hintChanged(ctx context.Context, a access) {
	if err := publishChanged(ctx, h.deps.Pool, a); err != nil {
		h.deps.Log.WarnContext(ctx, "training: publish event", "err", err)
	}
}
