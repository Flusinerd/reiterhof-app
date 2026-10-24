package horses_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files/filestest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestHorsePhoto(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	save := func(stable string, data []byte, ct string) files.Saved {
		t.Helper()
		s, err := files.Save(ctx, stable, bytes.NewReader(data), ct)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	first := save(seed.StableB, filestest.PNG(), "image/png")
	second := save(seed.StableB, filestest.JPEG(), "image/jpeg")
	pdf := save(seed.StableB, []byte("%PDF-1.7\nhello\n"), "application/pdf")
	foreign := save(otherStable, filestest.PNG(), "image/png")
	path := "/api/v1/horses/" + seed.HorseLuna

	// No photo yet: photo_url is null, the file is served to nobody.
	rec := e.call(seed.UserTom, "GET", path, nil)
	e.expect(rec, 200)
	if got := decode[horses.Horse](t, rec); got.PhotoURL != nil {
		t.Fatalf("photo_url = %v, want nil", *got.PhotoURL)
	}
	e.expect(e.call(seed.UserTom, "GET", first.URL, nil), 404)

	// Only owner and admin may set it; invalid paths are refused.
	body := map[string]any{"photo_path": first.Path}
	e.expect(e.call(seed.UserTom, "PATCH", path, body), 403)
	e.expect(e.call(seed.UserMia, "PATCH", path, body), 403)
	e.expect(e.call(otherUser, "PATCH", path, body), 404)
	for _, bad := range []string{foreign.Path, pdf.Path, "../../etc/passwd", seed.StableB + "/" + strings.Repeat("0", 32) + ".png"} {
		e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"photo_path": bad}), 400)
	}

	rec = e.call(seed.UserJan, "PATCH", path, body)
	e.expect(rec, 200)
	got := decode[horses.Horse](t, rec)
	if got.PhotoURL == nil || *got.PhotoURL != first.URL {
		t.Fatalf("photo_url = %v, want %s", got.PhotoURL, first.URL)
	}
	// The list carries it too, and every member of the stable may load the file.
	for _, h := range decode[[]horses.Horse](t, e.call(seed.UserTom, "GET", "/api/v1/horses", nil)) {
		if (h.ID == seed.HorseLuna) != (h.PhotoURL != nil) {
			t.Errorf("%s photo_url = %v", h.Name, h.PhotoURL)
		}
	}
	for u, want := range map[string]int{seed.UserTom: 200, seed.UserMia: 200, seed.UserAnna: 200, otherUser: 404, "": 401} {
		if rec := e.call(u, "GET", first.URL, nil); rec.Code != want {
			t.Errorf("photo as %q = %d, want %d", u, rec.Code, want)
		}
	}

	// Patching other fields leaves the photo alone; setting the same path keeps the file.
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"box": "9"}), 200)
	e.expect(e.call(seed.UserJan, "PATCH", path, body), 200)
	if _, _, err := files.Open(seed.StableB, first.Path); err != nil {
		t.Fatalf("photo after no-op patches: %v", err)
	}

	// Replacing removes the old file.
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"photo_path": second.Path}), 200)
	if _, _, err := files.Open(seed.StableB, first.Path); err != files.ErrNotFound {
		t.Errorf("replaced photo: %v, want ErrNotFound", err)
	}
	e.expect(e.call(seed.UserTom, "GET", second.URL, nil), 200)

	// A file that another record still references survives being cleared.
	if _, err := e.pool.Exec(ctx, `INSERT INTO blankets (stable_id, horse_id, name, photo_path) VALUES ($1, $2, 'Decke', $3)`,
		seed.StableB, seed.HorseLuna, second.Path); err != nil {
		t.Fatal(err)
	}
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"photo_path": ""}), 200)
	if _, _, err := files.Open(seed.StableB, second.Path); err != nil {
		t.Errorf("shared photo was removed: %v", err)
	}

	// Clearing without other references removes the file and the URL.
	third := save(seed.StableB, filestest.PNG(), "image/png")
	e.expect(e.call(seed.UserAnna, "PATCH", path, map[string]any{"photo_path": third.Path}), 403)
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"photo_path": third.Path}), 200)
	rec = e.call(seed.UserJan, "PATCH", path, map[string]any{"photo_path": ""})
	e.expect(rec, 200)
	if got := decode[horses.Horse](t, rec); got.PhotoURL != nil {
		t.Errorf("photo_url after clearing = %v", *got.PhotoURL)
	}
	if _, _, err := files.Open(seed.StableB, third.Path); err != files.ErrNotFound {
		t.Errorf("cleared photo: %v, want ErrNotFound", err)
	}
	e.expect(e.call(seed.UserTom, "GET", third.URL, nil), 404)

	// A photo can be given on creation.
	fourth := save(seed.StableB, filestest.WebP(), "image/webp")
	rec = e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"name": "Neu", "photo_path": fourth.Path})
	e.expect(rec, 201)
	if got := decode[horses.Horse](t, rec); got.PhotoURL == nil || *got.PhotoURL != fourth.URL {
		t.Errorf("created photo_url = %v", got.PhotoURL)
	}
	e.expect(e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"name": "Neu2", "photo_path": foreign.Path}), 400)
}
