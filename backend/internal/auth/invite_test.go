package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"Jan@Example.ORG":   "jan@example.org",
		"  a.b@example.org": "a.b@example.org",
	} {
		got, ok := auth.NormalizeEmail(in)
		if !ok || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "no-at-sign", "Jan <jan@example.org>", "a@b@c"} {
		if _, ok := auth.NormalizeEmail(in); ok {
			t.Errorf("NormalizeEmail(%q) accepted", in)
		}
	}
}

func TestInviteCodeFormat(t *testing.T) {
	code, err := auth.NewInviteCode()
	if err != nil || len(code) != 8 || code != strings.ToUpper(code) {
		t.Fatalf("NewInviteCode = %q, %v", code, err)
	}
	if got := auth.FormatInviteCode("ABCDEFGH"); got != "ABCD-EFGH" {
		t.Errorf("FormatInviteCode = %q", got)
	}
}

func TestCreateInviteWithoutCreator(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	expires := time.Now().Add(24 * time.Hour)
	code, err := auth.CreateInvite(context.Background(), pool, seed.StableB, nil, expires, 3)
	if err != nil {
		t.Fatal(err)
	}
	var maxUses int
	var createdBy *string
	if err := pool.QueryRow(context.Background(),
		`SELECT max_uses, created_by::text FROM stable_invites WHERE code = $1 AND stable_id = $2`, code, seed.StableB).
		Scan(&maxUses, &createdBy); err != nil {
		t.Fatal(err)
	}
	if maxUses != 3 || createdBy != nil {
		t.Errorf("max_uses = %d, created_by = %v", maxUses, createdBy)
	}
}
