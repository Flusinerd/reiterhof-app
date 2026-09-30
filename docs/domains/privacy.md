# Privacy, imprint and consents (JAN-19)

Backend: `backend/internal/privacy`, migration `0110_privacy.up.sql`. Mobile: `app/settings/privacy.tsx`,
`app/legal/{privacy,imprint}.tsx`, `lib/consent*.ts`, `lib/api/privacy.ts`, `components/consent-*.tsx`.
Texts: `docs/legal/` (German drafts, **Entwurf – vor Veröffentlichung rechtlich prüfen lassen**).
What the owner must sign and fill in: [docs/legal/avv-checkliste.md](../legal/avv-checkliste.md).

## Consents

Table `consents(user_id, stable_id, kind, version, granted_at, revoked_at)`, unique per `(user_id, kind)`:
a grant upserts the row (new version and time, `revoked_at` cleared), a revocation sets `revoked_at`. A kind
without a row was never granted. `stable_id` is nullable (a person may consent before joining a stable).

| Kind | Meaning | Server-side effect |
| --- | --- | --- |
| `location_geofence` | background location for "Automatisch erkennen" | `POST /presence/check-in` with `source: "geofence"` is refused with `403 consent_required` without a grant (`internal/presence`) |
| `location_tracking` | recording the GPS track of a training session | none yet: the tracking feature checks `privacy.Has(ctx, pool, userID, privacy.KindLocationTracking)` |
| `presence_sharing` | others may see that I am at the stable | revoking sets `users.presence_visibility` to `hidden` |
| `photos` | camera and photo library | none (device permission plus explanation) |
| `push` | push notifications | revoking deletes the user's `push_tokens` |

`privacy.TextVersion` (`"2026-09-30"`) is the version of the privacy text; a grant stores it. Changing the texts
means raising both `TextVersion` and the `**Textversion:**` line in `docs/legal/*.md` (a test checks this); existing
grants then show `up_to_date: false` and the app asks again, but nothing is locked.

| Route (signed-in user, stable **not** required) | Notes |
| --- | --- |
| `GET /api/v1/me/consents` | `{current_version, items: [{kind, granted, version, granted_at, revoked_at, current_version, up_to_date}]}`, always all five kinds in the order above |
| `PUT /api/v1/me/consents/{kind}` | body `{granted: bool, version?: string}`; `version` is the text the app shows, a different one on a grant gives `409 version_mismatch`; unknown kind `404`; returns the item |

For other packages: `privacy.Has(ctx, q, userID, kind)` tells whether a grant exists and is not revoked.

## Data subject rights

**Export** `GET /api/v1/me/export` (attachment `reiterhof-export-<date>.json`, `Cache-Control: no-store`). All
personal data of the caller across tables, as whole table rows (`to_jsonb`, so new columns appear automatically):
`profile`, `stable`, `sign_in_identities`, `sign_in_sessions` (no token hashes), `consents`, `push_tokens` (masked to
first and last four characters), `reminder_settings`, `reminders`, `presence_visits`, `training_sessions`
(including GPS tracks), `week_slots`, `observations_reported`, `requests_created`, `requests_helped`, `horses_owned`,
`horse_rider_roles`, `blanket_changes`, `reha_days_done`, `documents_uploaded` and `uploaded_files` (paths of documents
and observation photos). Only rows the user owns or authored; rows of other people are never included. **When you add a
table that references `users`, add a section to `privacy.Export` and to `privacy.DeleteAccount`.**

**Deletion** `POST /api/v1/me/delete {"confirm": true}` → `204`.

Decisions:

- **Owned horses block the deletion** (`409 owns_horses`, body lists `horses: [{id, name}]`). The alternative
  (keep the horse without owner) would leave the emergency card without a responsible person and the riders
  without someone who manages them. An admin changes the owner (`PATCH /horses/{id} owner_id`) or the horse is deleted;
  then deletion works.
- **The last admin of a stable is blocked** (`409 last_admin`), so a stable is never left without administration.
- **The `users` row is anonymised, not deleted**, because requests, training sessions, observations and week slots of a
  horse reference it (`NOT NULL` foreign keys) and the horse's history should survive: `name = "Gelöschtes Mitglied"`,
  `email = deleted-<id>@deleted.invalid` (the address can be registered again), `phone`, `avatar_color`, `stable_id`
  cleared, `is_admin = false`, `deleted_at` set. Without a stable the row does not show up in any member list.
- **Deleted:** sessions, sign-in identities, login tokens for the email, push tokens, reminder settings, reminders,
  presence visits, rider roles, helper assignments (a request that was `assigned` only to this user is `open` again),
  consents, GPS tracks (`sessions.track`) and raw gait windows (`gait_windows`), the media of reported observations (files are removed from disk after the
  commit; photos may show people and carry location metadata).
- **Kept, now attributed to "Gelöschtes Mitglied":** observation text, training sessions (without track), requests
  (open and assigned ones are `cancelled`), week slots, document metadata.
- Publishes `presence.changed` for the stable. Users without a stable can export and delete too.
- Backups still contain the data until they expire (14 daily, 8 weekly locally, 90 days offsite); documented in the
  privacy text. After a restore, repeat deletions.

## Retention

Scheduler job `privacy-retention`, daily 03:30 Europe/Berlin (`privacy.Job`, `privacy.Prune`; constants in
`retention.go`):

| Data | Kept | Then |
| --- | --- | --- |
| finished presence visits (`presence.left_at`) | 12 months | deleted (open visits are closed by `presence-close-stale` after 12 hours) |
| `sessions.track` (raw GPS) | 12 months after `started_at` | set to `NULL`, the session (duration, distance, gait shares) stays |
| `gait_windows` (raw gait features) | 12 months after the session's `started_at` | deleted |
| `reminders` | 12 months after `due_at` | deleted |
| `login_tokens` | until `expires_at` (15 minutes) | deleted when expired |
| `auth_sessions` | 90 days sliding | deleted when expired |
| Backups | 14 daily + 8 weekly on the server, 90 days offsite (`deploy/backup.sh`) | deleted |

Raw gait windows (`gait_windows`, stored with tracked sessions to improve the gait model) follow the track: deleted
12 months after the session's `started_at`, and immediately when the account is deleted. Everything else (horse record, documents, requests) is kept until a user with
the right deletes it or the account is deleted.

## Texts and how the app shows them

`docs/legal/datenschutz.md` and `docs/legal/impressum.md` are the **single source**. The app bundles them as
`mobile/lib/consent-legal-texts.ts`, generated by `node scripts/gen-legal.mjs` (the file says "GENERATED";
`lib/consent-legal.test.ts` fails while it is out of date, `--check` does the same on the command line). After editing
a text: run the script, raise `**Textversion:**` in both files and `privacy.TextVersion` together, commit all of it.
The texts use only headings (`#` to `###`), paragraphs, `- ` lists, `> ` quotes and `**bold**`
(`lib/consent-markdown.ts` renders exactly that; the test rejects tables, code and links).
The consent explanations of the sheets (short, plain German) are UI copy in `lib/consent-core.ts` (`CONSENT_COPY`).

## App

- `useConsentPrompt()` (`components/consent-prompt.tsx`): `if (!(await consent.ensure("photos"))) return;` shows the
  explanation sheet ("Erlauben", "Nicht jetzt", "Mehr erfahren" → `/legal/privacy`) when the consent is missing,
  revoked or for an old text. Render `{consent.sheet}` in the screen. Gated so far: the geofence switch and "Bin da"
  (`presence_sharing`, declining sets the visibility to hidden) on the presence screen, camera and photo library in the
  horse documents screen, push registration (`useDeviceSetup` registers the token only with the consent; the first-run
  sheet comes from `ConsentOnboarding` in the root layout, asked once per device).
  **New features that record location, take photos or send notifications must call `ensure` first.**
- `app/settings/privacy.tsx`: switch per consent (revoking the geofence consent also stops it on the device), data
  export (JSON file written to the cache and shared with the system share sheet), links to the texts, "Konto
  löschen" with a confirmation. Reachable from the presence screen and from the join screen (no stable needed).
- `app/legal/privacy.tsx`, `app/legal/imprint.tsx`: reachable without a session (links on sign-in and join).

## Tests

`go test ./internal/privacy ./internal/presence` (needs `REITERHOF_TEST_DATABASE_URL`): consent grant, revoke,
re-grant, version conflict, cross-user isolation, consequences of revocation, geofence check-in refused without
consent, export contains only own data (also for the other stable), deletion (anonymised row, data gone, other users
and the other stable untouched, files removed, requests reopened/cancelled), blocked deletion for owners and last
admins, retention, and a test that the legal texts carry `TextVersion` and the draft marker. Mobile:
`lib/consent-core.test.ts`, `lib/consent-markdown.test.ts`, `lib/consent-legal.test.ts`.
