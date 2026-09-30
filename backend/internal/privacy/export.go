package privacy

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"time"
)

// filePathRe finds upload paths (<stable uuid>/<32 hex>.<ext>, see internal/files) in any JSON.
var filePathRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/[0-9a-f]{32}\.(?:jpg|png|webp|heic|pdf)`)

// maskToken hides most of a push token: the first and last four characters remain.
func maskToken(t string) string {
	if len(t) <= 12 {
		return "****"
	}
	return t[:4] + "..." + t[len(t)-4:]
}

// rowsJSON runs a query whose single column is jsonb and returns the values (never nil).
func rowsJSON(ctx context.Context, q DB, sql string, args ...any) ([]json.RawMessage, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(raw))
	}
	return out, rows.Err()
}

// Export collects all personal data of the user across tables (Art. 15 and 20 GDPR).
//
// Rows are exported as JSON documents of the whole table row (to_jsonb), so new columns
// appear automatically. Only the user's own records are included: authored or owned rows,
// never rows of other people. Sessions are listed without their token hash, push tokens
// are masked.
func Export(ctx context.Context, q DB, userID string, now time.Time) (map[string]any, error) {
	doc := map[string]any{"exported_at": now.UTC(), "text_version": TextVersion}

	type section struct {
		key string
		sql string
	}
	sections := []section{
		{"profile", `SELECT to_jsonb(u) FROM users u WHERE u.id = $1`},
		{"stable", `SELECT jsonb_build_object('id', s.id, 'name', s.name, 'city', s.city)
			FROM users u JOIN stables s ON s.id = u.stable_id WHERE u.id = $1`},
		{"sign_in_identities", `SELECT jsonb_build_object('provider', provider, 'email', email, 'created_at', created_at)
			FROM auth_identities WHERE user_id = $1 ORDER BY created_at`},
		{"sign_in_sessions", `SELECT jsonb_build_object('created_at', created_at, 'last_seen_at', last_seen_at,
				'expires_at', expires_at, 'user_agent', user_agent)
			FROM auth_sessions WHERE user_id = $1 ORDER BY created_at`},
		{"reminder_settings", `SELECT jsonb_build_object('kind', kind, 'enabled', enabled) FROM reminder_settings
			WHERE user_id = $1 ORDER BY kind`},
		{"reminders", `SELECT to_jsonb(r) FROM reminders r WHERE r.user_id = $1 ORDER BY r.due_at`},
		{"presence_visits", `SELECT to_jsonb(p) FROM presence p WHERE p.user_id = $1 ORDER BY p.arrived_at`},
		{"training_sessions", `SELECT to_jsonb(s) FROM sessions s WHERE s.user_id = $1 ORDER BY s.started_at`},
		{"week_slots", `SELECT to_jsonb(w) FROM week_slots w WHERE w.user_id = $1 ORDER BY w.day`},
		{"observations_reported", `SELECT to_jsonb(o) FROM observations o WHERE o.reported_by = $1 ORDER BY o.created_at`},
		{"requests_created", `SELECT to_jsonb(r) FROM requests r WHERE r.created_by = $1 ORDER BY r.date, r.created_at`},
		{"requests_helped", `SELECT jsonb_build_object('request_id', r.id, 'type', r.type, 'date', r.date, 'status', r.status,
				'horse_id', r.horse_id, 'thanked', a.thanked, 'since', a.created_at)
			FROM request_assignees a JOIN requests r ON r.id = a.request_id
			WHERE a.user_id = $1 ORDER BY r.date, a.created_at`},
		{"horses_owned", `SELECT to_jsonb(h) FROM horses h WHERE h.owner_id = $1 ORDER BY h.name`},
		{"horse_rider_roles", `SELECT jsonb_build_object('horse_id', hr.horse_id, 'horse_name', h.name, 'rules', hr.rules,
				'since', hr.created_at)
			FROM horse_riders hr JOIN horses h ON h.id = hr.horse_id WHERE hr.user_id = $1 ORDER BY h.name`},
		{"blanket_changes", `SELECT to_jsonb(b) FROM blanket_states b WHERE b.changed_by = $1 ORDER BY b.changed_at`},
		{"reha_days_done", `SELECT to_jsonb(d) FROM reha_days d WHERE d.done_by = $1 ORDER BY d.day`},
		{"documents_uploaded", `SELECT jsonb_build_object('id', id, 'horse_id', horse_id, 'kind', kind, 'title', title,
				'file_path', file_path, 'created_at', created_at)
			FROM horse_documents WHERE uploaded_by = $1 ORDER BY created_at`},
	}
	for _, s := range sections {
		rows, err := rowsJSON(ctx, q, s.sql, userID)
		if err != nil {
			return nil, err
		}
		if s.key == "profile" || s.key == "stable" {
			if len(rows) == 0 {
				doc[s.key] = nil
			} else {
				doc[s.key] = rows[0]
			}
			continue
		}
		doc[s.key] = rows
	}

	consents, err := List(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	doc["consents"] = consents

	tokens := []map[string]any{}
	rows, err := q.Query(ctx, `SELECT token, platform, created_at FROM push_tokens WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var token, platform string
		var created time.Time
		if err := rows.Scan(&token, &platform, &created); err != nil {
			rows.Close()
			return nil, err
		}
		tokens = append(tokens, map[string]any{"token": maskToken(token), "platform": platform, "created_at": created})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	doc["push_tokens"] = tokens

	files, err := uploadedFiles(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	doc["uploaded_files"] = files
	return doc, nil
}

type fileRef struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

// uploadedFiles lists the files the user attached: documents they uploaded and the media of
// observations they reported. The files themselves are not part of the JSON; the app serves
// them under /api/v1/files/<path> while the user has access.
func uploadedFiles(ctx context.Context, q DB, userID string) ([]fileRef, error) {
	out := []fileRef{}
	docs, err := rowsJSON(ctx, q, `SELECT to_jsonb(file_path) FROM horse_documents WHERE uploaded_by = $1`, userID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		var p string
		if json.Unmarshal(d, &p) == nil {
			out = append(out, fileRef{Path: p, Source: "horse_documents"})
		}
	}
	paths, err := observationMediaPaths(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		out = append(out, fileRef{Path: p, Source: "observations"})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// observationMediaPaths returns the upload paths mentioned in the media of the user's
// observations, whatever shape the JSON has.
func observationMediaPaths(ctx context.Context, q DB, userID string) ([]string, error) {
	rows, err := rowsJSON(ctx, q, `SELECT media FROM observations WHERE reported_by = $1`, userID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range rows {
		for _, p := range filePathRe.FindAllString(string(raw), -1) {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out, nil
}
